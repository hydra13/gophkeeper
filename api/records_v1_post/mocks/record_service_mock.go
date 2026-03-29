package mocks

import (
	"context"
	"sync"

	"github.com/hydra13/gophkeeper/api/records_v1_post"
	"github.com/hydra13/gophkeeper/internal/models"
)

// RecordServiceMock implements recordsv1post.RecordService
type RecordServiceMock struct {
	mu  sync.Mutex
	err error

	CreateRecordCalled bool
	LastRecord        *models.Record
}

// NewRecordServiceMock creates a new mock.
func NewRecordServiceMock(err error) *RecordServiceMock {
	return &RecordServiceMock{err: err}
}

// CreateRecord implements recordsv1post.RecordService.
func (m *RecordServiceMock) CreateRecord(record *models.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateRecordCalled = true
	m.LastRecord = record
	return m.err
}

// Assert interface compliance.
var _ recordsv1post.RecordService = (*RecordServiceMock)(nil)

// unused import guard
var _ context.Context
