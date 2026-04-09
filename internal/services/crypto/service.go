//go:generate minimock -i .KeyProvider -o mocks -s _mock.go -g
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

const (
	encryptedPrefix = "GK1"
	nonceSize       = 12
)

var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// KeyProvider возвращает ключ шифрования по версии.
type KeyProvider interface {
	KeyForEncrypt(version int64) ([]byte, error)
	KeyForDecrypt(version int64) ([]byte, error)
}

// CryptoService описывает операции шифрования payload.
type CryptoService interface {
	Encrypt(data []byte, keyVersion int64) ([]byte, error)
	Decrypt(data []byte, keyVersion int64) ([]byte, error)
}

// Service шифрует данные с префиксом версии ключа.
type Service struct {
	keys KeyProvider
	rand io.Reader
}

// New создаёт сервис шифрования.
func New(keys KeyProvider) *Service {
	return &Service{keys: keys, rand: rand.Reader}
}

func (s *Service) Encrypt(data []byte, keyVersion int64) ([]byte, error) {
	if s == nil || s.keys == nil {
		return nil, errors.New("key provider is required")
	}
	key, err := s.keys.KeyForEncrypt(keyVersion)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(s.rand, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	result := make([]byte, 0, len(encryptedPrefix)+len(nonce)+len(ciphertext))
	result = append(result, encryptedPrefix...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func (s *Service) Decrypt(data []byte, keyVersion int64) ([]byte, error) {
	if s == nil || s.keys == nil {
		return nil, errors.New("key provider is required")
	}
	if len(data) < len(encryptedPrefix)+nonceSize+1 {
		return nil, ErrInvalidCiphertext
	}
	if string(data[:len(encryptedPrefix)]) != encryptedPrefix {
		return nil, ErrInvalidCiphertext
	}
	key, err := s.keys.KeyForDecrypt(keyVersion)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceStart := len(encryptedPrefix)
	nonceEnd := nonceStart + nonceSize
	nonce := data[nonceStart:nonceEnd]
	ciphertext := data[nonceEnd:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// HasEncryptedPrefix проверяет наличие префикса версии ключа в данных.
func HasEncryptedPrefix(data []byte) bool {
	return len(data) >= len(encryptedPrefix) && string(data[:len(encryptedPrefix)]) == encryptedPrefix
}
