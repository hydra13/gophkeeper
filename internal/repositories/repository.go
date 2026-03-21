package repositories

import "gophkeeper/internal/models"

// Repository интерфейс для хранилища данных
type Repository interface {
	UserRepository
	DataRepository
	SyncRepository
}

type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByLogin(login string) (*models.User, error)
	GetUserByID(id int64) (*models.User, error)
}

type DataRepository interface {
	// LoginPassword operations
	CreateLoginPassword(data *models.LoginPassword) error
	GetLoginPassword(id int64) (*models.LoginPassword, error)
	ListLoginPasswords(userID int64) ([]models.LoginPassword, error)
	UpdateLoginPassword(data *models.LoginPassword) error
	DeleteLoginPassword(id int64) error

	// TextData operations
	CreateTextData(data *models.TextData) error
	GetTextData(id int64) (*models.TextData, error)
	ListTextData(userID int64) ([]models.TextData, error)
	UpdateTextData(data *models.TextData) error
	DeleteTextData(id int64) error

	// BinaryData operations
	CreateBinaryData(data *models.BinaryData) error
	GetBinaryData(id int64) (*models.BinaryData, error)
	ListBinaryData(userID int64) ([]models.BinaryData, error)
	UpdateBinaryData(data *models.BinaryData) error
	DeleteBinaryData(id int64) error

	// BankCard operations
	CreateBankCard(data *models.BankCard) error
	GetBankCard(id int64) (*models.BankCard, error)
	ListBankCards(userID int64) ([]models.BankCard, error)
	UpdateBankCard(data *models.BankCard) error
	DeleteBankCard(id int64) error
}

type SyncRepository interface {
	GetSyncRecords(userID int64, since string) ([]models.SyncRecord, error)
	CreateSyncRecord(record *models.SyncRecord) error
	UpdateSyncRecord(record *models.SyncRecord) error
}
