package mocks

import "github.com/hydra13/gophkeeper/internal/models"

// SyncPullerMock — мок SyncPuller для тестов.
type SyncPullerMock struct {
	PullFunc func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error)
}

// Pull вызывает мок-реализацию PullFunc.
func (m *SyncPullerMock) Pull(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
	return m.PullFunc(userID, deviceID, cursor, limit)
}
