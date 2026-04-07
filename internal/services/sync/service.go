//go:generate minimock -i .SyncRepo,.RecordRepo -o mocks -s _mock.go -g
package sync

import (
	"errors"
	"fmt"
	"iter"

	"github.com/hydra13/gophkeeper/internal/models"
)

type RecordRepo interface {
	CreateRecord(record *models.Record) error
	GetRecord(id int64) (*models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64) error
}

type SyncRepo interface {
	GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error)
	CreateRevision(rev *models.RecordRevision) error
	GetMaxRevision(userID int64) (int64, error)
	GetConflicts(userID int64) ([]models.SyncConflict, error)
	CreateConflict(conflict *models.SyncConflict) error
	ResolveConflict(conflictID int64, resolution string) error
}

type revisionsSeqRepo interface {
	GetRevisionsSeq(userID int64, sinceRevision int64) iter.Seq2[models.RecordRevision, error]
}

// Service координирует pull/push синхронизацию и конфликты.
type Service struct {
	syncRepo   SyncRepo
	recordRepo RecordRepo
}

// NewService создаёт сервис синхронизации.
func NewService(syncRepo SyncRepo, recordRepo RecordRepo) (*Service, error) {
	if syncRepo == nil {
		return nil, errors.New("sync repository is required")
	}
	if recordRepo == nil {
		return nil, errors.New("record repository is required")
	}
	return &Service{syncRepo: syncRepo, recordRepo: recordRepo}, nil
}

// Push применяет локальные изменения клиента.
func (s *Service) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
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

func (s *Service) pushChange(userID int64, deviceID string, change models.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	if change.Deleted {
		return s.pushDelete(userID, deviceID, change)
	}
	if change.Record == nil || change.Record.ID == 0 {
		return s.pushCreate(userID, deviceID, change)
	}
	return s.pushUpdate(userID, deviceID, change)
}

func (s *Service) pushDelete(userID int64, deviceID string, change models.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
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

func (s *Service) pushCreate(userID int64, deviceID string, change models.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	nextRev, err := s.nextRevision(userID)
	if err != nil {
		return nil, nil, err
	}

	record := &models.Record{
		UserID:         userID,
		Type:           change.Record.Type,
		Name:           change.Record.Name,
		Metadata:       change.Record.Metadata,
		Revision:       nextRev,
		DeviceID:       deviceID,
		KeyVersion:     change.Record.KeyVersion,
		PayloadVersion: change.Record.PayloadVersion,
		Payload:        change.Record.Payload,
	}

	if err := record.Validate(); err != nil {
		return nil, nil, err
	}

	if err := s.recordRepo.CreateRecord(record); err != nil {
		return nil, nil, fmt.Errorf("create record: %w", err)
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

func (s *Service) pushUpdate(userID int64, deviceID string, change models.PendingChange) (*models.RecordRevision, *models.SyncConflict, error) {
	existing, err := s.recordRepo.GetRecord(change.Record.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get record for update: %w", err)
	}
	if existing.UserID != userID {
		return nil, nil, models.ErrRecordNotFound
	}

	if change.BaseRevision != existing.Revision {
		conflict, err := s.createConflict(userID, existing, change)
		if err != nil {
			return nil, nil, err
		}
		return nil, conflict, nil
	}

	if existing.IsDeleted() {
		if err := existing.Restore(); err != nil {
			return nil, nil, err
		}
	}

	existing.Name = change.Record.Name
	existing.Metadata = change.Record.Metadata
	existing.DeviceID = deviceID
	existing.Payload = change.Record.Payload
	if change.Record.KeyVersion > 0 {
		existing.KeyVersion = change.Record.KeyVersion
	}
	if change.Record.PayloadVersion > 0 {
		existing.PayloadVersion = change.Record.PayloadVersion
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

func (s *Service) createConflict(userID int64, serverRecord *models.Record, change models.PendingChange) (*models.SyncConflict, error) {
	localRecord := &models.Record{
		UserID:         userID,
		Type:           change.Record.Type,
		Name:           change.Record.Name,
		Metadata:       change.Record.Metadata,
		DeviceID:       change.Record.DeviceID,
		KeyVersion:     change.Record.KeyVersion,
		PayloadVersion: change.Record.PayloadVersion,
		Payload:        change.Record.Payload,
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

func (s *Service) createDeleteConflict(userID int64, serverRecord *models.Record, change models.PendingChange) (*models.SyncConflict, error) {
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

// Pull возвращает изменения и конфликты начиная с указанной ревизии.
func (s *Service) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	if limit <= 0 {
		limit = 50
	}

	revisions, records, err := s.collectPullRecords(userID, sinceRevision, limit)
	if err != nil {
		return nil, nil, nil, err
	}

	conflicts, err := s.syncRepo.GetConflicts(userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get conflicts: %w", err)
	}

	return revisions, records, conflicts, nil
}

func (s *Service) collectPullRecords(userID int64, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, error) {
	if seqRepo, ok := s.syncRepo.(revisionsSeqRepo); ok {
		return s.collectPullRecordsSeq(userID, limit, seqRepo.GetRevisionsSeq(userID, sinceRevision))
	}

	revisions, err := s.syncRepo.GetRevisions(userID, sinceRevision)
	if err != nil {
		return nil, nil, fmt.Errorf("get revisions: %w", err)
	}
	if int64(len(revisions)) > limit {
		revisions = revisions[:limit]
	}
	records, err := s.recordsForRevisions(revisions)
	if err != nil {
		return nil, nil, err
	}
	return revisions, records, nil
}

func (s *Service) collectPullRecordsSeq(userID int64, limit int64, revisionsSeq iter.Seq2[models.RecordRevision, error]) ([]models.RecordRevision, []models.Record, error) {
	revisions := make([]models.RecordRevision, 0, limit)
	recordByID := make(map[int64]bool)
	var records []models.Record

	for revision, err := range revisionsSeq {
		if err != nil {
			return nil, nil, fmt.Errorf("get revisions: %w", err)
		}
		if int64(len(revisions)) >= limit {
			break
		}
		revisions = append(revisions, revision)
		if recordByID[revision.RecordID] {
			continue
		}
		recordByID[revision.RecordID] = true
		record, err := s.recordRepo.GetRecord(revision.RecordID)
		if err != nil {
			return nil, nil, fmt.Errorf("get record %d: %w", revision.RecordID, err)
		}
		records = append(records, *record)
	}

	return revisions, records, nil
}

func (s *Service) recordsForRevisions(revisions []models.RecordRevision) ([]models.Record, error) {
	recordByID := make(map[int64]bool, len(revisions))
	var records []models.Record
	for _, rev := range revisions {
		if recordByID[rev.RecordID] {
			continue
		}
		recordByID[rev.RecordID] = true
		rec, err := s.recordRepo.GetRecord(rev.RecordID)
		if err != nil {
			return nil, fmt.Errorf("get record %d: %w", rev.RecordID, err)
		}
		records = append(records, *rec)
	}
	return records, nil
}

// GetConflicts возвращает незакрытые конфликты пользователя.
func (s *Service) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return s.syncRepo.GetConflicts(userID)
}

// ResolveConflict разрешает конфликт и возвращает итоговую запись.
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
