package repositories

import "github.com/hydra13/gophkeeper/internal/models"

// Repository объединяет все контракты слоя хранения.
type Repository interface {
	UserRepository
	RecordRepository
	SyncRepository
	SessionRepository
	UploadRepository
	KeyVersionRepository
}

// BlobStorage хранит бинарные данные вне основной БД.
type BlobStorage interface {
	Save(path string, data []byte) error
	Read(path string) ([]byte, error)
	Delete(path string) error
	Exists(path string) (bool, error)
}

// UserRepository работает с пользователями.
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
}

// RecordRepository работает с записями.
type RecordRepository interface {
	CreateRecord(record *models.Record) error
	GetRecord(id int64) (*models.Record, error)
	ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	UpdateRecord(record *models.Record) error
	DeleteRecord(id int64) error
}

// SyncRepository хранит ревизии и конфликты синхронизации.
type SyncRepository interface {
	GetRevisions(userID int64, sinceRevision int64) ([]models.RecordRevision, error)
	CreateRevision(rev *models.RecordRevision) error
	GetMaxRevision(userID int64) (int64, error)
	GetConflicts(userID int64) ([]models.SyncConflict, error)
	CreateConflict(conflict *models.SyncConflict) error
	ResolveConflict(conflictID int64, resolution string) error
}

// SessionRepository работает с пользовательскими сессиями.
type SessionRepository interface {
	CreateSession(session *models.Session) error
	GetSession(id int64) (*models.Session, error)
	GetSessionByRefreshToken(token string) (*models.Session, error)
	RevokeSession(id int64) error
	RevokeSessionsByUser(userID int64) error
	UpdateLastSeenAt(id int64) error
}

// UploadRepository хранит состояние загрузок и чанков.
type UploadRepository interface {
	CreateUploadSession(session *models.UploadSession) error
	GetUploadSession(id int64) (*models.UploadSession, error)
	GetCompletedUploadByRecordID(recordID int64) (*models.UploadSession, error)
	UpdateUploadSession(session *models.UploadSession) error
	SaveChunk(chunk *models.Chunk) error
	GetChunks(uploadID int64) ([]models.Chunk, error)
}

// KeyVersionRepository работает с версиями ключей шифрования.
type KeyVersionRepository interface {
	CreateKeyVersion(kv *models.KeyVersion) error
	GetKeyVersion(version int64) (*models.KeyVersion, error)
	GetActiveKeyVersion() (*models.KeyVersion, error)
	ListKeyVersions() ([]models.KeyVersion, error)
	UpdateKeyVersion(kv *models.KeyVersion) error
}
