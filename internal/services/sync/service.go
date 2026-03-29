package sync

import (
	"errors"
	"fmt"

	"github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories"
)

// Service реализует бизнес-логику синхронизации.
type Service struct {
	syncRepo   repositories.SyncRepository
	recordRepo repositories.RecordRepository
}

// NewService создаёт новый sync service.
func NewService(syncRepo repositories.SyncRepository, recordRepo repositories.RecordRepository) (*Service, error) {
	if syncRepo == nil {
		return nil, errors.New("sync repository is required")
	}
	if recordRepo == nil {
		return nil, errors.New("record repository is required")
	}
	return &Service{syncRepo: syncRepo, recordRepo: recordRepo}, nil
}

// Push обрабатывает локальные изменения от клиента.
func (s *Service) Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	var accepted []models.RecordRevision
	var conflicts []models.SyncConflict

	for _, change := range changes {
		rev, conflict, err := s.pushChange(userID, deviceID, change)
		if err != nil {
			return nil, nil, err
		}
		if rev != nil {
			accepted = append(accepted, *rev)
		}
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	return accepted, conflicts, nil
}

func (s *Service) pushChange(userID int64, deviceID string, change sync_push_v1_post.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	if change.Deleted {
		return s.pushDelete(userID, deviceID, change)
	}
	if change.Record.ID == 0 {
		return s.pushCreate(userID, deviceID, change)
	}
	return s.pushUpdate(userID, deviceID, change)
}

func (s *Service) pushDelete(userID int64, deviceID string, change sync_push_v1_post.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	recordID := change.Record.ID
	record, err := s.recordRepo.GetRecord(recordID)
	if err != nil {
		return nil, nil, fmt.Errorf("get record for delete: %w", err)
	}
	if record.UserID != userID {
		return nil, nil, models.ErrRecordNotFound
	}
	if record.IsDeleted() {
		return nil, nil, nil
	}

	// Конфликт: base_revision клиента не совпадает с текущей ревизией сервера
	if change.BaseRevision != 0 && change.BaseRevision != record.Revision {
		conflict, err := s.createDeleteConflict(userID, record, change)
		if err != nil {
			return nil, nil, err
		}
		return nil, conflict, nil
	}

	if err := s.recordRepo.DeleteRecord(recordID); err != nil {
		return nil, nil, fmt.Errorf("delete record: %w", err)
	}

	nextRev, err := s.nextRevision(userID)
	if err != nil {
		return nil, nil, err
	}

	rev := &models.RecordRevision{
		RecordID: recordID,
		UserID:   userID,
		Revision: nextRev,
		DeviceID: deviceID,
	}
	if err := s.syncRepo.CreateRevision(rev); err != nil {
		return nil, nil, fmt.Errorf("create revision: %w", err)
	}

	return rev, nil, nil
}

func (s *Service) pushCreate(userID int64, deviceID string, change sync_push_v1_post.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	dto := change.Record
	record := &models.Record{
		UserID:         userID,
		Type:           models.RecordType(dto.Type),
		Name:           dto.Name,
		Metadata:       dto.Metadata,
		DeviceID:       deviceID,
		KeyVersion:     dto.KeyVersion,
		PayloadVersion: dto.PayloadVersion,
		Payload:        dtoPayloadToDomain(dto),
	}

	if err := record.Validate(); err != nil {
		return nil, nil, err
	}

	if err := s.recordRepo.CreateRecord(record); err != nil {
		return nil, nil, fmt.Errorf("create record: %w", err)
	}

	nextRev, err := s.nextRevision(userID)
	if err != nil {
		return nil, nil, err
	}

	rev := &models.RecordRevision{
		RecordID: record.ID,
		UserID:   userID,
		Revision: nextRev,
		DeviceID: deviceID,
	}
	if err := s.syncRepo.CreateRevision(rev); err != nil {
		return nil, nil, fmt.Errorf("create revision: %w", err)
	}

	return rev, nil, nil
}

func (s *Service) pushUpdate(userID int64, deviceID string, change sync_push_v1_post.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	dto := change.Record
	existing, err := s.recordRepo.GetRecord(dto.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get record for update: %w", err)
	}
	if existing.UserID != userID {
		return nil, nil, models.ErrRecordNotFound
	}

	// Конфликт: base_revision клиента не совпадает с текущей ревизией сервера
	if change.BaseRevision != existing.Revision {
		conflict, err := s.createConflict(userID, existing, change)
		if err != nil {
			return nil, nil, err
		}
		return nil, conflict, nil
	}

	// Восстановление soft-deleted записи: если запись удалена, а клиент присылает update с актуальной ревизией
	if existing.IsDeleted() {
		if err := existing.Restore(); err != nil {
			return nil, nil, err
		}
	}

	// Принять изменение
	existing.Name = dto.Name
	existing.Metadata = dto.Metadata
	existing.DeviceID = deviceID
	existing.Payload = dtoPayloadToDomain(dto)
	if dto.KeyVersion > 0 {
		existing.KeyVersion = dto.KeyVersion
	}
	if dto.PayloadVersion > 0 {
		existing.PayloadVersion = dto.PayloadVersion
	}

	nextRev, err := s.nextRevision(userID)
	if err != nil {
		return nil, nil, err
	}
	if err := existing.BumpRevision(nextRev, deviceID); err != nil {
		return nil, nil, err
	}

	if err := s.recordRepo.UpdateRecord(existing); err != nil {
		return nil, nil, fmt.Errorf("update record: %w", err)
	}

	rev := &models.RecordRevision{
		RecordID: existing.ID,
		UserID:   userID,
		Revision: nextRev,
		DeviceID: deviceID,
	}
	if err := s.syncRepo.CreateRevision(rev); err != nil {
		return nil, nil, fmt.Errorf("create revision: %w", err)
	}

	return rev, nil, nil
}

func (s *Service) createConflict(userID int64, serverRecord *models.Record, change sync_push_v1_post.PendingChange) (*models.SyncConflict, error) {
	localRecord := &models.Record{
		UserID:         userID,
		Type:           models.RecordType(change.Record.Type),
		Name:           change.Record.Name,
		Metadata:       change.Record.Metadata,
		DeviceID:       change.Record.DeviceID,
		KeyVersion:     change.Record.KeyVersion,
		PayloadVersion: change.Record.PayloadVersion,
		Payload:        dtoPayloadToDomain(change.Record),
	}

	conflict := &models.SyncConflict{
		UserID:         userID,
		RecordID:       serverRecord.ID,
		LocalRevision:  change.BaseRevision,
		ServerRevision: serverRecord.Revision,
		LocalRecord:    localRecord,
		ServerRecord:   serverRecord,
	}
	if err := s.syncRepo.CreateConflict(conflict); err != nil {
		return nil, fmt.Errorf("create conflict: %w", err)
	}
	return conflict, nil
}

func (s *Service) createDeleteConflict(userID int64, serverRecord *models.Record, change sync_push_v1_post.PendingChange) (*models.SyncConflict, error) {
	conflict := &models.SyncConflict{
		UserID:         userID,
		RecordID:       serverRecord.ID,
		LocalRevision:  change.BaseRevision,
		ServerRevision: serverRecord.Revision,
		ServerRecord:   serverRecord,
	}
	if err := s.syncRepo.CreateConflict(conflict); err != nil {
		return nil, fmt.Errorf("create delete conflict: %w", err)
	}
	return conflict, nil
}

// Pull возвращает изменения для пользователя начиная с указанного курсора.
func (s *Service) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	if limit <= 0 {
		limit = 50
	}

	revisions, err := s.syncRepo.GetRevisions(userID, sinceRevision)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get revisions: %w", err)
	}

	if int64(len(revisions)) > limit {
		revisions = revisions[:limit]
	}

	recordByID := make(map[int64]bool, len(revisions))
	var records []models.Record
	for _, rev := range revisions {
		if recordByID[rev.RecordID] {
			continue
		}
		recordByID[rev.RecordID] = true
		rec, err := s.recordRepo.GetRecord(rev.RecordID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("get record %d: %w", rev.RecordID, err)
		}
		records = append(records, *rec)
	}

	conflicts, err := s.syncRepo.GetConflicts(userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get conflicts: %w", err)
	}

	return revisions, records, conflicts, nil
}

// GetConflicts возвращает нерешённые конфликты.
func (s *Service) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return s.syncRepo.GetConflicts(userID)
}

// ResolveConflict разрешает конфликт и обновляет запись.
func (s *Service) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	conflicts, err := s.syncRepo.GetConflicts(userID)
	if err != nil {
		return nil, err
	}

	var conflict *models.SyncConflict
	for i := range conflicts {
		if conflicts[i].ID == conflictID {
			conflict = &conflicts[i]
			break
		}
	}
	if conflict == nil {
		return nil, models.ErrRecordNotFound
	}

	if err := conflict.Resolve(resolution); err != nil {
		return nil, err
	}

	if err := s.syncRepo.ResolveConflict(conflictID, resolution); err != nil {
		return nil, err
	}

	var record *models.Record
	switch resolution {
	case models.ConflictResolutionLocal:
		if conflict.LocalRecord != nil {
			record = conflict.LocalRecord
			record.ID = conflict.RecordID
			record.UserID = userID
			nextRev, err := s.nextRevision(userID)
			if err != nil {
				return nil, err
			}
			if err := record.BumpRevision(nextRev, record.DeviceID); err != nil {
				return nil, err
			}
			if err := s.recordRepo.UpdateRecord(record); err != nil {
				return nil, fmt.Errorf("update record with local: %w", err)
			}
			rev := &models.RecordRevision{
				RecordID: record.ID,
				UserID:   userID,
				Revision: nextRev,
				DeviceID: record.DeviceID,
			}
			if err := s.syncRepo.CreateRevision(rev); err != nil {
				return nil, fmt.Errorf("create revision: %w", err)
			}
		}
	case models.ConflictResolutionServer:
		record = conflict.ServerRecord
	default:
		return nil, models.ErrInvalidConflictResolution
	}

	return record, nil
}

func (s *Service) nextRevision(userID int64) (int64, error) {
	maxRev, err := s.syncRepo.GetMaxRevision(userID)
	if err != nil {
		return 0, err
	}
	return maxRev + 1, nil
}

func dtoPayloadToDomain(dto recordscommon.RecordDTO) models.RecordPayload {
	switch models.RecordType(dto.Type) {
	case models.RecordTypeLogin:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.LoginPayload{
				Login:    strVal(p["login"]),
				Password: strVal(p["password"]),
			}
		}
		return models.LoginPayload{}
	case models.RecordTypeText:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.TextPayload{Content: strVal(p["content"])}
		}
		return models.TextPayload{}
	case models.RecordTypeBinary:
		return models.BinaryPayload{}
	case models.RecordTypeCard:
		if p, ok := dto.Payload.(map[string]interface{}); ok {
			return models.CardPayload{
				Number:     strVal(p["number"]),
				HolderName: strVal(p["holder_name"]),
				ExpiryDate: strVal(p["expiry_date"]),
				CVV:        strVal(p["cvv"]),
			}
		}
		return models.CardPayload{}
	default:
		return nil
	}
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
