package mocks

import "github.com/hydra13/gophkeeper/internal/models"

// SyncPullerMock — мок SyncPuller для тестов.
type SyncPullerMock struct {
	PullFunc func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
}

// Pull вызывает мок-реализацию PullFunc.
func (m *SyncPullerMock) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return m.PullFunc(userID, deviceID, sinceRevision, limit)
}
