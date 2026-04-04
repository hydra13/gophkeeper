package models

import "errors"

// Ошибки аутентификации и пользователей.
var (
	// ErrUserNotFound — пользователь не найден.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials — некорректные учётные данные.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnauthorized — запрос без действительной аутентификации.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrSessionExpired — сессия пользователя истекла.
	ErrSessionExpired = errors.New("session expired")
	// ErrSessionRevoked — сессия пользователя отозвана.
	ErrSessionRevoked = errors.New("session revoked")
	// ErrEmailAlreadyExists — email уже зарегистрирован.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidUserID — некорректный идентификатор пользователя.
	ErrInvalidUserID = errors.New("invalid user id")
)

// Ошибки записей и данных.
var (
	// ErrRecordNotFound — запись не найдена.
	ErrRecordNotFound = errors.New("record not found")
	// ErrNilPayload — payload записи не задан.
	ErrNilPayload = errors.New("payload is required")
	// ErrEmptyRecordName — имя записи не задано.
	ErrEmptyRecordName = errors.New("record name is required")
	// ErrInvalidRecordType — некорректный тип записи.
	ErrInvalidRecordType = errors.New("invalid record type")
	// ErrRevisionNotMonotonic — новая ревизия не превышает текущую.
	ErrRevisionNotMonotonic = errors.New("revision must be greater than current")
	// ErrAlreadyDeleted — запись уже удалена (soft delete).
	ErrAlreadyDeleted = errors.New("record is already deleted")
	// ErrNotDeleted — запись не удалена (нельзя восстановить).
	ErrNotDeleted = errors.New("record is not deleted")
	// ErrEmptyDeviceID — device_id обязателен.
	ErrEmptyDeviceID = errors.New("device_id is required")
	// ErrInvalidKeyVersion — key_version должна быть > 0.
	ErrInvalidKeyVersion = errors.New("key_version must be positive")
	// ErrInvalidPayloadVersion — payload_version должна быть > 0 для binary записей.
	ErrInvalidPayloadVersion = errors.New("payload_version must be positive for binary records")
)

// Ошибки синхронизации и ревизий.
var (
	// ErrRevisionConflict — конфликт ревизий при синхронизации.
	ErrRevisionConflict = errors.New("revision conflict")
	// ErrConflictAlreadyResolved — конфликт уже разрешён.
	ErrConflictAlreadyResolved = errors.New("conflict already resolved")
	// ErrInvalidConflictResolution — некорректная стратегия разрешения конфликта.
	ErrInvalidConflictResolution = errors.New("invalid conflict resolution")
)

// Ошибки валидации банковских карт.
var (
	// ErrInvalidCardNumber — номер карты не проходит проверку (длина или Luhn).
	ErrInvalidCardNumber = errors.New("invalid card number")
	// ErrEmptyCardHolder — имя владельца карты не задано.
	ErrEmptyCardHolder = errors.New("card holder name is required")
	// ErrInvalidExpiryDate — некорректный формат expiry (ожидается MM/YY).
	ErrInvalidExpiryDate = errors.New("invalid expiry date, expected MM/YY")
	// ErrInvalidCVV — CVV должен содержать 3 или 4 цифры.
	ErrInvalidCVV = errors.New("invalid CVV, expected 3 or 4 digits")
)

// Ошибки загрузок (upload/download).
var (
	// ErrUploadNotFound — upload-сессия не найдена.
	ErrUploadNotFound = errors.New("upload session not found")
	// ErrUploadNotPending — upload-сессия не в состоянии pending.
	ErrUploadNotPending = errors.New("upload session is not pending")
	// ErrUploadCompleted — upload-сессия уже завершена.
	ErrUploadCompleted = errors.New("upload session already completed")
	// ErrUploadAborted — upload-сессия прервана.
	ErrUploadAborted = errors.New("upload session is aborted")
	// ErrChunkOutOfRange — индекс чанка вне допустимого диапазона.
	ErrChunkOutOfRange = errors.New("chunk index out of range")
	// ErrDuplicateChunk — чанк с таким индексом уже принят.
	ErrDuplicateChunk = errors.New("chunk already received")
	// ErrChunkOutOfOrder — нарушен ожидаемый порядок чанков.
	ErrChunkOutOfOrder = errors.New("chunk order violated")
)

// Ошибки скачивания (download).
var (
	// ErrDownloadNotFound — download-сессия не найдена.
	ErrDownloadNotFound = errors.New("download session not found")
	// ErrDownloadCompleted — download-сессия уже завершена.
	ErrDownloadCompleted = errors.New("download session already completed")
	// ErrDownloadAborted — download-сессия прервана.
	ErrDownloadAborted = errors.New("download session is aborted")
	// ErrDownloadNotActive — download-сессия не активна.
	ErrDownloadNotActive = errors.New("download session is not active")
	// ErrChunkAlreadyConfirmed — чанк уже подтверждён клиентом.
	ErrChunkAlreadyConfirmed = errors.New("chunk already confirmed by client")
)

// Ошибки ключей шифрования.
var (
	// ErrUnknownKeyVersion — неизвестная версия ключа шифрования.
	ErrUnknownKeyVersion = errors.New("unknown key version")
	// ErrKeyVersionNotActive — версия ключа не активна.
	ErrKeyVersionNotActive = errors.New("key version is not active")
	// ErrKeyVersionCannotDecrypt — версия ключа не подходит для расшифровки.
	ErrKeyVersionCannotDecrypt = errors.New("key version cannot decrypt")
)
