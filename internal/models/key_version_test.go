package models

import "testing"

func TestKeyVersionDeprecate(t *testing.T) {
	kv := &KeyVersion{Status: KeyStatusActive}
	if err := kv.Deprecate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.Status != KeyStatusDeprecated {
		t.Fatalf("expected deprecated status, got %s", kv.Status)
	}
	if kv.DeprecatedAt == nil {
		t.Fatal("expected DeprecatedAt to be set")
	}
}

func TestKeyVersionDeprecate_NotActive(t *testing.T) {
	kv := &KeyVersion{Status: KeyStatusDeprecated}
	if err := kv.Deprecate(); err == nil {
		t.Fatal("expected error when deprecating non-active key")
	}
}

func TestKeyVersionRetire(t *testing.T) {
	kv := &KeyVersion{Status: KeyStatusDeprecated}
	if err := kv.Retire(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.Status != KeyStatusRetired {
		t.Fatalf("expected retired status, got %s", kv.Status)
	}
	if kv.RetiredAt == nil {
		t.Fatal("expected RetiredAt to be set")
	}
}

func TestKeyVersionRetire_NotDeprecated(t *testing.T) {
	kv := &KeyVersion{Status: KeyStatusActive}
	if err := kv.Retire(); err == nil {
		t.Fatal("expected error when retiring non-deprecated key")
	}
}

func TestKeyVersionIsActive(t *testing.T) {
	kv := &KeyVersion{Status: KeyStatusActive}
	if !kv.IsActive() {
		t.Fatal("expected key to be active")
	}

	kv2 := &KeyVersion{Status: KeyStatusDeprecated}
	if kv2.IsActive() {
		t.Fatal("expected deprecated key to not be active")
	}
}

func TestKeyVersionCanDecrypt(t *testing.T) {
	active := &KeyVersion{Status: KeyStatusActive}
	deprecated := &KeyVersion{Status: KeyStatusDeprecated}
	retired := &KeyVersion{Status: KeyStatusRetired}

	if !active.CanDecrypt() {
		t.Fatal("active key should be able to decrypt")
	}
	if !deprecated.CanDecrypt() {
		t.Fatal("deprecated key should be able to decrypt")
	}
	if retired.CanDecrypt() {
		t.Fatal("retired key should not be able to decrypt")
	}
}

func TestKeyVersionFullLifecycle(t *testing.T) {
	kv := &KeyVersion{
		Version: 1,
		Status:  KeyStatusActive,
	}

	if !kv.IsActive() || !kv.CanDecrypt() {
		t.Fatal("new key should be active and able to decrypt")
	}

	_ = kv.Deprecate()
	if kv.IsActive() || !kv.CanDecrypt() {
		t.Fatal("deprecated key should not be active but should still decrypt")
	}

	_ = kv.Retire()
	if kv.IsActive() || kv.CanDecrypt() {
		t.Fatal("retired key should not be active and should not decrypt")
	}

	// Проверяем что DeprecatedAt установлен до RetiredAt
	if kv.DeprecatedAt == nil || kv.RetiredAt == nil {
		t.Fatal("both timestamps should be set")
	}
	if kv.DeprecatedAt.After(*kv.RetiredAt) {
		t.Fatal("deprecated should happen before retired")
	}
}
