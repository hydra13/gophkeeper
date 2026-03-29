package mocks

import (
	"github.com/hydra13/gophkeeper/internal/api/sync_push_v1_post"
	"github.com/hydra13/gophkeeper/internal/models"
)

// SyncPusherMock — мок SyncPusher для тестов.
type SyncPusherMock struct {
	PushFunc func(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

// Push вызывает мок-реализацию PushFunc.
func (m *SyncPusherMock) Push(userID int64, deviceID string, changes []sync_push_v1_post.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	return m.PushFunc(userID, deviceID, changes)
}
