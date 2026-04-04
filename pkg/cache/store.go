package cache

import (
	"github.com/hydra13/gophkeeper/internal/models"
)

// Store объединяет все разделы клиентского кеша.
type Store interface {
	Records() RecordCache

	Pending() PendingQueue

	Transfers() TransferState

	Auth() AuthStore

	Sync() SyncState

	Flush() error
}

// RecordCache хранит записи в локальном кеше.
type RecordCache interface {
	Get(id int64) (*models.Record, bool)

	GetAll() []models.Record

	Put(record *models.Record)

	PutAll(records []models.Record)

	Delete(id int64)

	Clear()
}

// OperationType задаёт тип отложенной операции.
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
)

// PendingOp описывает операцию, ожидающую синхронизации.
type PendingOp struct {
	ID           int64
	RecordID     int64
	Operation    OperationType
	Record       *models.Record
	BaseRevision int64
	CreatedAt    int64 // Unix timestamp
}

// PendingQueue хранит операции, ожидающие отправки на сервер.
type PendingQueue interface {
	Enqueue(op PendingOp) error

	DequeueAll() ([]PendingOp, error)

	Peek() ([]PendingOp, error)

	Len() int

	Clear()
}

// TransferType задаёт направление передачи бинарных данных.
type TransferType string

const (
	TransferUpload   TransferType = "upload"
	TransferDownload TransferType = "download"
)

// TransferStatus задаёт состояние передачи.
type TransferStatus string

const (
	TransferStatusActive    TransferStatus = "active"
	TransferStatusPaused    TransferStatus = "paused"
	TransferStatusCompleted TransferStatus = "completed"
)

// Transfer описывает незавершённую загрузку или скачивание.
type Transfer struct {
	ID           int64
	Type         TransferType
	RecordID     int64
	SessionID    int64 // upload_id или download_id
	TotalChunks  int64
	CompletedIdx int64 // индекс последнего обработанного чанка
	Status       TransferStatus
	ChunkSize    int64
	TotalSize    int64
	Data         []byte // для upload: данные, ожидающие отправки (только для маленьких файлов)
}

// TransferState хранит состояние незавершённых передач.
type TransferState interface {
	Save(t Transfer) error

	Get(id int64) (Transfer, bool)

	GetByRecord(recordID int64) (Transfer, bool)

	Delete(id int64)

	ListActive() []Transfer

	ListPending() []Transfer

	Clear()
}

// AuthData хранит локальное состояние авторизации.
type AuthData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	DeviceID     string `json:"device_id"`
}

// AuthStore хранит токены и сведения о текущем пользователе.
type AuthStore interface {
	Get() (*AuthData, bool)

	Set(data AuthData) error

	Clear()
}

// SyncData хранит состояние синхронизации клиента.
type SyncData struct {
	LastRevision int64 `json:"last_revision"`
}

// SyncState хранит последнюю полученную ревизию.
type SyncState interface {
	Get() SyncData

	SetLastRevision(rev int64) error
}
