package repositories

import "github.com/hydra13/gophkeeper/internal/models"

// Repository — объединённый интерфейс хранилища данных.
type Repository interface {
	UserRepository
	RecordRepository
	SyncRepository
	SessionRepository
	UploadRepository
	KeyVersionRepository
}

// BlobStorage — интерфейс хранения бинарных данных (payload/chunks).
type BlobStorage interface {
	Save(path string, data []byte) error
	Read(path string) ([]byte, error)
	Delete(path string) error
	Exists(path string) (bool, error)
}

// UserRepository — операции с пользователями.
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
}

// RecordRepository — операции с записями секретов.
type RecordRepository interface {
	CreateRecord(record *models.Record) error
	GetRecord(id int64) (*models.Record, error)
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64) error
}

// SyncRepository — операции с ревизиями и конфликтами синхронизации.
type SyncRepository interface {
	GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error)
	CreateRevision(rev *models.RecordRevision) error
	GetConflicts(userID int64) ([]models.SyncConflict, error)
	CreateConflict(conflict *models.SyncConflict) error
	ResolveConflict(conflictID int64, resolution string) error
}

// SessionRepository — операции с device-aware сессиями.
type SessionRepository interface {
	CreateSession(session *models.Session) error
	GetSession(id int64) (*models.Session, error)
	GetSessionByRefreshToken(token string) (*models.Session, error)
	RevokeSession(id int64) error
	RevokeSessionsByUser(userID int64) error
	UpdateLastSeenAt(id int64) error
}

// UploadRepository — операции с upload-сессиями и чанками.
type UploadRepository interface {
	CreateUploadSession(session *models.UploadSession) error
	GetUploadSession(id int64) (*models.UploadSession, error)
	GetCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error)
	UpdateUploadSession(session *models.UploadSession) error
	SaveChunk(chunk *models.Chunk) error
	GetChunks(uploadID int64) ([]models.Chunk, error)
}

// KeyVersionRepository — операции с версиями ключей шифрования.
type KeyVersionRepository interface {
	CreateKeyVersion(kv *models.KeyVersion) error
	GetKeyVersion(version int64) (*models.KeyVersion, error)
	GetActiveKeyVersion() (*models.KeyVersion, error)
	ListKeyVersions() ([]models.KeyVersion, error)
	UpdateKeyVersion(kv *models.KeyVersion) error
}
