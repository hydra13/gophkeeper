package records

import (
	"errors"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories"
)

// KeyManager предоставляет доступ к актуальной версии ключа.
type KeyManager interface {
	EnsureActive() (*models.KeyVersion, error)
}

// Service реализует бизнес-логику работы с записями.
type Service struct {
	repo repositories.RecordRepository
	keys KeyManager
}

// NewService создаёт новый сервис записей.
func NewService(repo repositories.RecordRepository, keys KeyManager) (*Service, error) {
	if repo == nil {
		return nil, errors.New("record repository is required")
	}
	if keys == nil {
		return nil, errors.New("key manager is required")
	}
	return &Service{repo: repo, keys: keys}, nil
}

// CreateRecord создаёт новую запись и привязывает её к активному key_version.
func (s *Service) CreateRecord(record *models.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}
	active, err := s.keys.EnsureActive()
	if err != nil {
		return err
	}
	record.KeyVersion = active.Version
	if err := record.Validate(); err != nil {
		return err
	}
	return s.repo.CreateRecord(record)
}

// ListRecords возвращает записи пользователя с опциональной фильтрацией.
// По умолчанию (recordType="", includeDeleted=false) — только активные записи всех типов.
func (s *Service) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return s.repo.ListRecords(userID, recordType, includeDeleted)
}

// GetRecord возвращает запись по ID.
// Возвращает ErrRecordNotFound для soft-deleted записей.
func (s *Service) GetRecord(id int64) (*models.Record, error) {
	record, err := s.repo.GetRecord(id)
	if err != nil {
		return nil, err
	}
	if record.IsDeleted() {
		return nil, models.ErrRecordNotFound
	}
	return record, nil
}

// UpdateRecord обновляет запись.
func (s *Service) UpdateRecord(record *models.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateRecord(record)
}

// DeleteRecord выполняет soft delete записи.
func (s *Service) DeleteRecord(id int64, deviceID string) error {
	if deviceID == "" {
		return models.ErrEmptyDeviceID
	}
	return s.repo.DeleteRecord(id)
}
