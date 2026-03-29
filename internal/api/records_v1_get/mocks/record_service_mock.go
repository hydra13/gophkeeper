package mocks

import (
	"github.com/hydra13/gophkeeper/internal/api/records_v1_get"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordServiceMock implements recordsv1get.RecordService
type RecordServiceMock struct {
	err     error
	records []models.Record
}

// NewRecordServiceMock creates a new mock.
func NewRecordServiceMock(records []models.Record, err error) *RecordServiceMock {
	return &RecordServiceMock{records: records, err: err}
}

// ListRecords implements recordsv1get.RecordService.
func (m *RecordServiceMock) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return m.records, m.err
}

// Assert interface compliance.
var _ recordsv1get.RecordService = (*RecordServiceMock)(nil)
