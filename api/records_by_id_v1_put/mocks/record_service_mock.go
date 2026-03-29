package mocks

import (
	"github.com/hydra13/gophkeeper/api/records_by_id_v1_put"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordServiceMock implements recordsbyidv1put.RecordService
type RecordServiceMock struct {
	record    *models.Record
	getErr    error
	updateErr error
}

// NewRecordServiceMock creates a new mock.
func NewRecordServiceMock(record *models.Record, getErr, updateErr error) *RecordServiceMock {
	return &RecordServiceMock{record: record, getErr: getErr, updateErr: updateErr}
}

// GetRecord implements recordsbyidv1put.RecordService.
func (m *RecordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.getErr
}

// UpdateRecord implements recordsbyidv1put.RecordService.
func (m *RecordServiceMock) UpdateRecord(record *models.Record) error {
	return m.updateErr
}

// Assert interface compliance.
var _ recordsbyidv1put.RecordService = (*RecordServiceMock)(nil)
