package mocks

import (
	"github.com/hydra13/gophkeeper/api/records_by_id_v1_get"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordServiceMock implements recordsbyidv1get.RecordService
type RecordServiceMock struct {
	record *models.Record
	err    error
}

// NewRecordServiceMock creates a new mock.
func NewRecordServiceMock(record *models.Record, err error) *RecordServiceMock {
	return &RecordServiceMock{record: record, err: err}
}

// GetRecord implements recordsbyidv1get.RecordService.
func (m *RecordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.err
}

// Assert interface compliance.
var _ recordsbyidv1get.RecordService = (*RecordServiceMock)(nil)
