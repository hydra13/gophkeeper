//go:generate minimock -i .RecordRepo,.KeyManager -o mocks -s _mock.go -g
package records

import (
	"errors"

	"github.com/hydra13/gophkeeper/internal/models"
)

type RecordRepo interface {
	CreateRecord(record *models.Record) error
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	GetRecord(id int64) (*models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64) error
}

type KeyManager interface {
	EnsureActive() (*models.KeyVersion, error)
}

// Service реализует операции создания, чтения и удаления записей.
type Service struct {
	repo RecordRepo
	keys KeyManager
}

// NewService создаёт сервис записей.
func NewService(repo RecordRepo, keys KeyManager) (*Service, error) {
	if repo == nil {
		return nil, errors.New("record repository is required")
	}
	if keys == nil {
		return nil, errors.New("key manager is required")
	}
	return &Service{repo: repo, keys: keys}, nil
}

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

func (s *Service) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return s.repo.ListRecords(userID, recordType, includeDeleted)
}

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

func (s *Service) UpdateRecord(record *models.Record) error {
	if record == nil {
		return errors.New("record is nil")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	return s.repo.UpdateRecord(record)
}

func (s *Service) DeleteRecord(id int64, deviceID string) error {
	if deviceID == "" {
		return models.ErrEmptyDeviceID
	}
	return s.repo.DeleteRecord(id)
}
