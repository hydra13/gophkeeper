package crypto

import (
	"crypto/rand"
	"errors"
	"testing"
)

type fakeKeys struct {
	keys map[int64][]byte
}

func (f *fakeKeys) KeyForEncrypt(version int64) ([]byte, error) {
	key, ok := f.keys[version]
	if !ok {
		return nil, errors.New("unknown key")
	}
	return key, nil
}

func (f *fakeKeys) KeyForDecrypt(version int64) ([]byte, error) {
	key, ok := f.keys[version]
	if !ok {
		return nil, errors.New("unknown key")
	}
	return key, nil
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand error: %v", err)
	}
	provider := &fakeKeys{keys: map[int64][]byte{1: key}}
	service := New(provider)

	plaintext := []byte("payload")
	ciphertext, err := service.Encrypt(plaintext, 1)
	if err != nil {
		t.Fatalf("encrypt error: %v", err)
	}
	if !HasEncryptedPrefix(ciphertext) {
		t.Fatal("expected encrypted prefix")
	}
	decrypted, err := service.Decrypt(ciphertext, 1)
	if err != nil {
		t.Fatalf("decrypt error: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}
