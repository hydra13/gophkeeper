package mocks

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/pkg/apiclient"
)

// MockTransport — мок Transport для тестирования clientcore.
type MockTransport struct {
	RegisterFunc              func(ctx context.Context, email, password string) (int64, error)
	LoginFunc                 func(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error)
	RefreshFunc               func(ctx context.Context, refreshToken string) (string, string, error)
	LogoutFunc                func(ctx context.Context) error
	CreateRecordFunc          func(ctx context.Context, record *models.Record) (*models.Record, error)
	GetRecordFunc             func(ctx context.Context, id int64) (*models.Record, error)
	ListRecordsFunc           func(ctx context.Context, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	UpdateRecordFunc          func(ctx context.Context, record *models.Record) (*models.Record, error)
	DeleteRecordFunc          func(ctx context.Context, id int64) error
	PullFunc                  func(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error)
	PushFunc                  func(ctx context.Context, changes []apiclient.PendingChange, deviceID string) (*apiclient.PushResult, error)
	CreateUploadSessionFunc   func(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
	UploadChunkFunc           func(ctx context.Context, uploadID, chunkIndex int64, data []byte) error
	GetUploadStatusFunc       func(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error)
	CreateDownloadSessionFunc func(ctx context.Context, recordID int64) (int64, int64, error)
	DownloadChunkFunc         func(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error)
	ConfirmChunkFunc          func(ctx context.Context, downloadID, chunkIndex int64) error
	GetDownloadStatusFunc     func(ctx context.Context, downloadID int64) (*apiclient.DownloadStatus, error)
	SetAccessTokenFunc        func(token string)

	accessToken string
	online      bool
}

func (m *MockTransport) Register(ctx context.Context, email, password string) (int64, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, email, password)
	}
	return 1, nil
}

func (m *MockTransport) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, email, password, deviceID, deviceName, clientType)
	}
	m.accessToken = "test-access-token"
	m.online = true
	return "test-access-token", "test-refresh-token", nil
}

func (m *MockTransport) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, refreshToken)
	}
	m.accessToken = "new-access-token"
	return "new-access-token", "new-refresh-token", nil
}

func (m *MockTransport) Logout(ctx context.Context) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx)
	}
	m.accessToken = ""
	m.online = false
	return nil
}

func (m *MockTransport) CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if m.CreateRecordFunc != nil {
		return m.CreateRecordFunc(ctx, record)
	}
	result := *record
	result.ID = 1
	result.Revision = 1
	return &result, nil
}

func (m *MockTransport) GetRecord(ctx context.Context, id int64) (*models.Record, error) {
	if m.GetRecordFunc != nil {
		return m.GetRecordFunc(ctx, id)
	}
	return &models.Record{
		ID:       id,
		UserID:   1,
		Type:     models.RecordTypeLogin,
		Name:     "test",
		Payload:  models.LoginPayload{Login: "user", Password: "pass"},
		Revision: 1,
		DeviceID: "test-device",
	}, nil
}

func (m *MockTransport) ListRecords(ctx context.Context, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	if m.ListRecordsFunc != nil {
		return m.ListRecordsFunc(ctx, recordType, includeDeleted)
	}
	return []models.Record{}, nil
}

func (m *MockTransport) UpdateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if m.UpdateRecordFunc != nil {
		return m.UpdateRecordFunc(ctx, record)
	}
	result := *record
	result.Revision++
	return &result, nil
}

func (m *MockTransport) DeleteRecord(ctx context.Context, id int64) error {
	if m.DeleteRecordFunc != nil {
		return m.DeleteRecordFunc(ctx, id)
	}
	return nil
}

func (m *MockTransport) Pull(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*apiclient.PullResult, error) {
	if m.PullFunc != nil {
		return m.PullFunc(ctx, sinceRevision, deviceID, limit)
	}
	return &apiclient.PullResult{NextRevision: sinceRevision}, nil
}

func (m *MockTransport) Push(ctx context.Context, changes []apiclient.PendingChange, deviceID string) (*apiclient.PushResult, error) {
	if m.PushFunc != nil {
		return m.PushFunc(ctx, changes, deviceID)
	}
	return &apiclient.PushResult{}, nil
}

func (m *MockTransport) CreateUploadSession(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	if m.CreateUploadSessionFunc != nil {
		return m.CreateUploadSessionFunc(ctx, recordID, totalChunks, chunkSize, totalSize, keyVersion)
	}
	return 1, nil
}

func (m *MockTransport) UploadChunk(ctx context.Context, uploadID, chunkIndex int64, data []byte) error {
	if m.UploadChunkFunc != nil {
		return m.UploadChunkFunc(ctx, uploadID, chunkIndex, data)
	}
	return nil
}

func (m *MockTransport) GetUploadStatus(ctx context.Context, uploadID int64) (*apiclient.UploadStatus, error) {
	if m.GetUploadStatusFunc != nil {
		return m.GetUploadStatusFunc(ctx, uploadID)
	}
	return &apiclient.UploadStatus{UploadID: uploadID, Status: "PENDING"}, nil
}

func (m *MockTransport) CreateDownloadSession(ctx context.Context, recordID int64) (int64, int64, error) {
	if m.CreateDownloadSessionFunc != nil {
		return m.CreateDownloadSessionFunc(ctx, recordID)
	}
	return 1, 2, nil
}

func (m *MockTransport) DownloadChunk(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error) {
	if m.DownloadChunkFunc != nil {
		return m.DownloadChunkFunc(ctx, downloadID, chunkIndex)
	}
	return []byte("chunk-data"), nil
}

func (m *MockTransport) ConfirmChunk(ctx context.Context, downloadID, chunkIndex int64) error {
	if m.ConfirmChunkFunc != nil {
		return m.ConfirmChunkFunc(ctx, downloadID, chunkIndex)
	}
	return nil
}

func (m *MockTransport) GetDownloadStatus(ctx context.Context, downloadID int64) (*apiclient.DownloadStatus, error) {
	if m.GetDownloadStatusFunc != nil {
		return m.GetDownloadStatusFunc(ctx, downloadID)
	}
	return &apiclient.DownloadStatus{DownloadID: downloadID, Status: "ACTIVE"}, nil
}

func (m *MockTransport) SetAccessToken(token string) {
	m.accessToken = token
	if m.SetAccessTokenFunc != nil {
		m.SetAccessTokenFunc(token)
	}
}
