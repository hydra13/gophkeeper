package models

import (
	"fmt"
	"time"
)

// RecordType определяет тип записи.
type RecordType string

// Поддерживаемые типы записей.
const (
	RecordTypeLogin  RecordType = "login"
	RecordTypeText   RecordType = "text"
	RecordTypeBinary RecordType = "binary"
	RecordTypeCard   RecordType = "card"
)

// ValidRecordTypes содержит допустимые типы записей.
var ValidRecordTypes = map[RecordType]bool{
	RecordTypeLogin:  true,
	RecordTypeText:   true,
	RecordTypeBinary: true,
	RecordTypeCard:   true,
}

// Record описывает основную доменную сущность секрета.
type Record struct {
	ID             int64
	UserID         int64
	Type           RecordType
	Name           string
	Metadata       string
	Payload        RecordPayload
	Revision       int64
	DeletedAt      *time.Time
	DeviceID       string
	KeyVersion     int64
	PayloadVersion int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RecordPayload объединяет типизированные payload записей.
type RecordPayload interface {
	RecordType() RecordType
}

// Validate проверяет корректность записи перед сохранением.
func (r *Record) Validate() error {
	if !ValidRecordTypes[r.Type] {
		return fmt.Errorf("invalid record type: %s", r.Type)
	}
	if r.UserID <= 0 {
		return ErrInvalidUserID
	}
	if r.Name == "" {
		return ErrEmptyRecordName
	}
	if r.DeviceID == "" {
		return ErrEmptyDeviceID
	}
	if r.KeyVersion <= 0 {
		return ErrInvalidKeyVersion
	}
	if r.Payload == nil {
		return ErrNilPayload
	}
	if r.Payload.RecordType() != r.Type {
		return fmt.Errorf("payload type %s does not match record type %s", r.Payload.RecordType(), r.Type)
	}
	if r.Type == RecordTypeBinary && r.PayloadVersion <= 0 {
		return ErrInvalidPayloadVersion
	}
	return nil
}

// IsDeleted сообщает, что запись помечена удалённой.
func (r *Record) IsDeleted() bool {
	return r.DeletedAt != nil
}

// BumpRevision обновляет ревизию записи и устройство-источник изменения.
func (r *Record) BumpRevision(newRevision int64, deviceID string) error {
	if newRevision <= r.Revision {
		return ErrRevisionNotMonotonic
	}
	if deviceID == "" {
		return ErrEmptyDeviceID
	}
	r.Revision = newRevision
	r.DeviceID = deviceID
	r.UpdatedAt = time.Now()
	return nil
}

// SoftDelete помечает запись удалённой.
func (r *Record) SoftDelete() error {
	if r.IsDeleted() {
		return ErrAlreadyDeleted
	}
	now := time.Now()
	r.DeletedAt = &now
	r.UpdatedAt = now
	return nil
}

// Restore снимает пометку удаления с записи.
func (r *Record) Restore() error {
	if !r.IsDeleted() {
		return ErrNotDeleted
	}
	r.DeletedAt = nil
	r.UpdatedAt = time.Now()
	return nil
}
