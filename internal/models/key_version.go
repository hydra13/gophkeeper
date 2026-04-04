package models

import (
	"fmt"
	"time"
)

// KeyStatus описывает жизненный цикл версии ключа.
type KeyStatus string

const (
	KeyStatusActive     KeyStatus = "active"
	KeyStatusDeprecated KeyStatus = "deprecated"
	KeyStatusRetired    KeyStatus = "retired"
)

// KeyVersion хранит серверную версию ключа шифрования.
type KeyVersion struct {
	ID           int64
	Version      int64
	EncryptedKey []byte
	KeyNonce     []byte
	Status       KeyStatus
	CreatedAt    time.Time
	DeprecatedAt *time.Time
	RetiredAt    *time.Time
}

// Deprecate переводит активный ключ в состояние deprecated.
func (kv *KeyVersion) Deprecate() error {
	if kv.Status != KeyStatusActive {
		return fmt.Errorf("only active key can be deprecated, current status: %s", kv.Status)
	}
	now := time.Now()
	kv.Status = KeyStatusDeprecated
	kv.DeprecatedAt = &now
	return nil
}

// Retire переводит deprecated-ключ в состояние retired.
func (kv *KeyVersion) Retire() error {
	if kv.Status != KeyStatusDeprecated {
		return fmt.Errorf("only deprecated key can be retired, current status: %s", kv.Status)
	}
	now := time.Now()
	kv.Status = KeyStatusRetired
	kv.RetiredAt = &now
	return nil
}

// IsActive сообщает, что версия ключа активна.
func (kv *KeyVersion) IsActive() bool {
	return kv.Status == KeyStatusActive
}

// CanDecrypt сообщает, что ключ ещё можно использовать для расшифровки.
func (kv *KeyVersion) CanDecrypt() bool {
	return kv.Status == KeyStatusActive || kv.Status == KeyStatusDeprecated
}
