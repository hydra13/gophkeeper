package models

import (
	"fmt"
	"time"
)

// KeyStatus определяет статус версии ключа шифрования.
type KeyStatus string

const (
	// KeyStatusActive — ключ используется для шифрования новых данных.
	KeyStatusActive KeyStatus = "active"
	// KeyStatusDeprecated — ключ устарел, новые данные им не шифруются, но старые ещё расшифровываются.
	KeyStatusDeprecated KeyStatus = "deprecated"
	// KeyStatusRetired — ключ выведен из обращения, данные перешифрованы.
	KeyStatusRetired KeyStatus = "retired"
)

// KeyVersion — версия серверного ключа шифрования (master-key + data encryption keys).
type KeyVersion struct {
	// ID — уникальный идентификатор версии ключа.
	ID int64
	// Version — порядковый номер версии ключа.
	Version int64
	// Status — текущий статус ключа (active, deprecated, retired).
	Status KeyStatus
	// CreatedAt — время создания версии ключа.
	CreatedAt time.Time
	// DeprecatedAt — время перевода в статус deprecated.
	DeprecatedAt *time.Time
	// RetiredAt — время перевода в статус retired.
	RetiredAt *time.Time
}

// Deprecate переводит ключ в статус deprecated.
func (kv *KeyVersion) Deprecate() error {
	if kv.Status != KeyStatusActive {
		return fmt.Errorf("only active key can be deprecated, current status: %s", kv.Status)
	}
	now := time.Now()
	kv.Status = KeyStatusDeprecated
	kv.DeprecatedAt = &now
	return nil
}

// Retire переводит ключ в статус retired.
func (kv *KeyVersion) Retire() error {
	if kv.Status != KeyStatusDeprecated {
		return fmt.Errorf("only deprecated key can be retired, current status: %s", kv.Status)
	}
	now := time.Now()
	kv.Status = KeyStatusRetired
	kv.RetiredAt = &now
	return nil
}

// IsActive проверяет, является ли ключ активным.
func (kv *KeyVersion) IsActive() bool {
	return kv.Status == KeyStatusActive
}

// CanDecrypt проверяет, можно ли использовать ключ для расшифровки.
func (kv *KeyVersion) CanDecrypt() bool {
	return kv.Status == KeyStatusActive || kv.Status == KeyStatusDeprecated
}
