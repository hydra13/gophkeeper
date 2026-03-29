package mocks

import (
	"github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_delete"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordServiceMock implements recordsbyidv1delete.RecordService
type RecordServiceMock struct {
	record    *models.Record
	getErr    error
	deleteErr error
}

// NewRecordServiceMock creates a new mock.
func NewRecordServiceMock(record *models.Record, getErr, deleteErr error) *RecordServiceMock {
	return &RecordServiceMock{record: record, getErr: getErr, deleteErr: deleteErr}
}

// GetRecord implements recordsbyidv1delete.RecordService.
func (m *RecordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.getErr
}

// DeleteRecord implements recordsbyidv1delete.RecordService.
func (m *RecordServiceMock) DeleteRecord(id int64) error {
	return m.deleteErr
}

// Assert interface compliance.
var _ recordsbyidv1delete.RecordService = (*RecordServiceMock)(nil)
