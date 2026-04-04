package models

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionRevoked     = errors.New("session revoked")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidUserID      = errors.New("invalid user id")
)

var (
	ErrRecordNotFound        = errors.New("record not found")
	ErrNilPayload            = errors.New("payload is required")
	ErrEmptyRecordName       = errors.New("record name is required")
	ErrInvalidRecordType     = errors.New("invalid record type")
	ErrRevisionNotMonotonic  = errors.New("revision must be greater than current")
	ErrAlreadyDeleted        = errors.New("record is already deleted")
	ErrNotDeleted            = errors.New("record is not deleted")
	ErrEmptyDeviceID         = errors.New("device_id is required")
	ErrInvalidKeyVersion     = errors.New("key_version must be positive")
	ErrInvalidPayloadVersion = errors.New("payload_version must be positive for binary records")
)

var (
	ErrRevisionConflict          = errors.New("revision conflict")
	ErrConflictAlreadyResolved   = errors.New("conflict already resolved")
	ErrInvalidConflictResolution = errors.New("invalid conflict resolution")
)

var (
	ErrInvalidCardNumber = errors.New("invalid card number")
	ErrEmptyCardHolder   = errors.New("card holder name is required")
	ErrInvalidExpiryDate = errors.New("invalid expiry date, expected MM/YY")
	ErrInvalidCVV        = errors.New("invalid CVV, expected 3 or 4 digits")
)

var (
	ErrUploadNotFound   = errors.New("upload session not found")
	ErrUploadNotPending = errors.New("upload session is not pending")
	ErrUploadCompleted  = errors.New("upload session already completed")
	ErrUploadAborted    = errors.New("upload session is aborted")
	ErrChunkOutOfRange  = errors.New("chunk index out of range")
	ErrDuplicateChunk   = errors.New("chunk already received")
	ErrChunkOutOfOrder  = errors.New("chunk order violated")
)

var (
	ErrDownloadNotFound      = errors.New("download session not found")
	ErrDownloadCompleted     = errors.New("download session already completed")
	ErrDownloadAborted       = errors.New("download session is aborted")
	ErrDownloadNotActive     = errors.New("download session is not active")
	ErrChunkAlreadyConfirmed = errors.New("chunk already confirmed by client")
)

var (
	ErrUnknownKeyVersion       = errors.New("unknown key version")
	ErrKeyVersionNotActive     = errors.New("key version is not active")
	ErrKeyVersionCannotDecrypt = errors.New("key version cannot decrypt")
)
