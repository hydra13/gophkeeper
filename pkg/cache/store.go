package cache

import (
	"github.com/hydra13/gophkeeper/internal/models"
)

// Store — локальный кеш клиента: metadata записей, payload, pending-операции
// и состояние незавершённых upload/download.
// Потокобезопасный. Персистентность через JSON-файлы.
type Store interface {
	// Records — кеш метаданных и payload записей.
	Records() RecordCache

	// Pending — очередь операций, ожидающих отправки на сервер.
	Pending() PendingQueue

	// Transfers — состояние незавершённых upload/download.
	Transfers() TransferState

	// Auth — кеш токенов авторизации.
	Auth() AuthStore

	// Sync — состояние синхронизации (последняя ревизия).
	Sync() SyncState

	// Flush сохраняет все данные на диск.
	Flush() error
}

// RecordCache — кеш записей (metadata + payload).
type RecordCache interface {
	// Get возвращает запись из кеша по ID.
	Get(id int64) (*models.Record, bool)

	// GetAll возвращает все записи из кеша.
	GetAll() []models.Record

	// Put сохраняет запись в кеш.
	Put(record *models.Record)

	// PutAll заменяет весь кеш записей (после pull).
	PutAll(records []models.Record)

	// Delete удаляет запись из кеша.
	Delete(id int64)

	// Clear очищает весь кеш записей.
	Clear()
}

// OperationType — тип pending-операции.
type OperationType string

const (
	// OperationCreate — создание записи.
	OperationCreate OperationType = "create"
	// OperationUpdate — обновление записи.
	OperationUpdate OperationType = "update"
	// OperationDelete — удаление записи.
	OperationDelete OperationType = "delete"
)

// PendingOp — операция, ожидающая отправки на сервер.
type PendingOp struct {
	ID           int64
	RecordID     int64
	Operation    OperationType
	Record       *models.Record
	BaseRevision int64
	CreatedAt    int64 // Unix timestamp
}

// PendingQueue — очередь pending-операций.
type PendingQueue interface {
	// Enqueue добавляет операцию в очередь.
	Enqueue(op PendingOp) error

	// DequeueAll возвращает все pending-операции и очищает очередь.
	DequeueAll() ([]PendingOp, error)

	// Peek возвращает все pending-операции без удаления.
	Peek() ([]PendingOp, error)

	// Len возвращает количество pending-операций.
	Len() int

	// Clear очищает очередь.
	Clear()
}

// TransferType — тип передачи данных.
type TransferType string

const (
	// TransferUpload — загрузка данных на сервер.
	TransferUpload TransferType = "upload"
	// TransferDownload — скачивание данных с сервера.
	TransferDownload TransferType = "download"
)

// TransferStatus — статус передачи.
type TransferStatus string

const (
	// TransferStatusActive — передача в процессе.
	TransferStatusActive TransferStatus = "active"
	// TransferStatusPaused — передача приостановлена.
	TransferStatusPaused TransferStatus = "paused"
	// TransferStatusCompleted — передача завершена.
	TransferStatusCompleted TransferStatus = "completed"
)

// Transfer — незавершённая передача upload/download.
type Transfer struct {
	ID             int64
	Type           TransferType
	RecordID       int64
	SessionID      int64 // upload_id или download_id
	TotalChunks    int64
	CompletedIdx   int64 // индекс последнего обработанного чанка
	Status         TransferStatus
	ChunkSize      int64
	TotalSize      int64
	Data           []byte // для upload: данные, ожидающие отправки (только для маленьких файлов)
}

// TransferState — состояние незавершённых передач.
type TransferState interface {
	// Save сохраняет/обновляет transfer.
	Save(t Transfer) error

	// Get возвращает transfer по ID.
	Get(id int64) (Transfer, bool)

	// GetByRecord возвращает transfer по recordID.
	GetByRecord(recordID int64) (Transfer, bool)

	// Delete удаляет transfer.
	Delete(id int64)

	// ListActive возвращает все активные передачи.
	ListActive() []Transfer

	// ListPending возвращает все незавершённые передачи (active или paused).
	ListPending() []Transfer

	// Clear очищает все передачи.
	Clear()
}

// AuthData — кешированные данные авторизации.
type AuthData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	DeviceID     string `json:"device_id"`
}

// AuthStore — хранилище токенов авторизации.
type AuthStore interface {
	// Get возвращает кешированные данные авторизации.
	Get() (*AuthData, bool)

	// Set сохраняет данные авторизации.
	Set(data AuthData) error

	// Clear очищает данные авторизации.
	Clear()
}

// SyncData — состояние синхронизации.
type SyncData struct {
	LastRevision int64 `json:"last_revision"`
}

// SyncState — состояние последней синхронизации.
type SyncState interface {
	// Get возвращает текущее состояние синхронизации.
	Get() SyncData

	// SetLastRevision обновляет последнюю ревизию.
	SetLastRevision(rev int64) error
}
