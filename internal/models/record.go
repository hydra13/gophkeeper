// Package models содержит доменные сущности GophKeeper.
//
// Модели не зависят от транспортного слоя (HTTP, gRPC) и слоя хранения (PostgreSQL, файловое хранилище).
// Все инварианты и бизнес-правила зафиксированы на уровне домена.
package models

import (
	"fmt"
	"time"
)

// RecordType определяет тип хранимой записи секрета.
type RecordType string

const (
	// RecordTypeLogin — пара логин/пароль.
	RecordTypeLogin RecordType = "login"
	// RecordTypeText — произвольные текстовые данные.
	RecordTypeText RecordType = "text"
	// RecordTypeBinary — произвольные бинарные данные.
	RecordTypeBinary RecordType = "binary"
	// RecordTypeCard — данные банковской карты.
	RecordTypeCard RecordType = "card"
)

// ValidRecordTypes содержит все допустимые типы записей.
var ValidRecordTypes = map[RecordType]bool{
	RecordTypeLogin:  true,
	RecordTypeText:   true,
	RecordTypeBinary: true,
	RecordTypeCard:   true,
}

// Record — единая доменная сущность хранения секрета.
// Объединяет все типы (login, text, binary, card) через типизированный payload.
type Record struct {
	// ID — уникальный идентификатор записи.
	ID int64
	// UserID — владелец записи.
	UserID int64
	// Type — тип секрета (login, text, binary, card).
	Type RecordType
	// Name — пользовательское название записи.
	Name string
	// Metadata — произвольная текстовая метаинформация.
	Metadata string
	// Payload — типизированные данные в зависимости от Type.
	Payload RecordPayload
	// Revision — монотонно возрастающая версия записи для синхронизации.
	Revision int64
	// DeletedAt — время soft delete; nil означает, что запись активна.
	DeletedAt *time.Time
	// DeviceID — идентификатор устройства, создавшего или изменившего запись.
	DeviceID string
	// KeyVersion — версия серверного ключа шифрования, которым зашифрована запись.
	KeyVersion int64
	// PayloadVersion — версия payload (для бинарных данных).
	PayloadVersion int64
	// CreatedAt — время создания записи.
	CreatedAt time.Time
	// UpdatedAt — время последнего обновления записи.
	UpdatedAt time.Time
}

// RecordPayload — интерфейс типизированных данных записи.
// Реализации: LoginPayload, TextPayload, BinaryPayload, CardPayload.
type RecordPayload interface {
	// RecordType возвращает тип записи, к которому относится payload.
	RecordType() RecordType
}

// Validate проверяет корректность записи.
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

// IsDeleted проверяет, удалена ли запись (soft delete).
func (r *Record) IsDeleted() bool {
	return r.DeletedAt != nil
}

// BumpRevision увеличивает ревизию записи с проверкой монотонного роста.
// Возвращает ошибку, если newRevision не больше текущей ревизии.
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

// SoftDelete помечает запись как удалённую.
// Возвращает ошибку, если запись уже удалена.
func (r *Record) SoftDelete() error {
	if r.IsDeleted() {
		return ErrAlreadyDeleted
	}
	now := time.Now()
	r.DeletedAt = &now
	r.UpdatedAt = now
	return nil
}

// Restore восстанавливает удалённую запись.
// Возвращает ошибку, если запись не была удалена.
func (r *Record) Restore() error {
	if !r.IsDeleted() {
		return ErrNotDeleted
	}
	r.DeletedAt = nil
	r.UpdatedAt = time.Now()
	return nil
}
