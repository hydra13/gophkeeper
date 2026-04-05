package apiclient

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/models"
)

// Transport описывает операции удалённого сервиса, доступные клиенту.
type Transport interface {
	Register(ctx context.Context, email, password string) (userID int64, err error)
	Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (accessToken, refreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error)
	Logout(ctx context.Context) error

	CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error)
	GetRecord(ctx context.Context, id int64) (*models.Record, error)
	ListRecords(ctx context.Context, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	UpdateRecord(ctx context.Context, record *models.Record) (*models.Record, error)
	DeleteRecord(ctx context.Context, id int64, deviceID string) error

	Pull(ctx context.Context, sinceRevision int64, deviceID string, limit int32) (*PullResult, error)
	Push(ctx context.Context, changes []PendingChange, deviceID string) (*PushResult, error)

	CreateUploadSession(ctx context.Context, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (uploadID int64, err error)
	UploadChunk(ctx context.Context, uploadID, chunkIndex int64, data []byte) error
	GetUploadStatus(ctx context.Context, uploadID int64) (*UploadStatus, error)
	CreateDownloadSession(ctx context.Context, recordID int64) (downloadID int64, totalChunks int64, err error)
	DownloadChunk(ctx context.Context, downloadID, chunkIndex int64) ([]byte, error)
	ConfirmChunk(ctx context.Context, downloadID, chunkIndex int64) error
	GetDownloadStatus(ctx context.Context, downloadID int64) (*DownloadStatus, error)

	SetAccessToken(token string)
}

// PullResult содержит результат pull-синхронизации.
type PullResult struct {
	Records      []models.Record
	HasMore      bool
	NextRevision int64
	Conflicts    []SyncConflictInfo
}

// SyncConflictInfo описывает конфликт синхронизации на транспортном уровне.
type SyncConflictInfo struct {
	ID             int64
	RecordID       int64
	LocalRevision  int64
	ServerRevision int64
	Resolved       bool
	Resolution     string
	LocalRecord    *models.Record
	ServerRecord   *models.Record
}

// PendingChange описывает локальное изменение для push-синхронизации.
type PendingChange struct {
	Record       *models.Record
	Deleted      bool
	BaseRevision int64
}

// PushResult содержит результат отправки локальных изменений.
type PushResult struct {
	Accepted  []AcceptedChange
	Conflicts []SyncConflictInfo
}

// AcceptedChange описывает изменение, принятое сервером.
type AcceptedChange struct {
	RecordID int64
	Revision int64
	DeviceID string
}

// UploadStatus описывает состояние upload-сессии.
type UploadStatus struct {
	UploadID       int64
	Status         string
	TotalChunks    int64
	ReceivedChunks int64
	MissingChunks  []int64
}

// DownloadStatus описывает состояние download-сессии.
type DownloadStatus struct {
	DownloadID      int64
	Status          string
	TotalChunks     int64
	ConfirmedChunks int64
	RemainingChunks []int64
}
