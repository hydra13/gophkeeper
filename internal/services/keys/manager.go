//go:generate minimock -i .Repository -o mocks -s _mock.go -g
package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/hydra13/gophkeeper/internal/models"
)

const (
	dataKeySize  = 32
	nonceSizeGCM = 12
)

// Repository описывает операции хранения версий ключей.
type Repository interface {
	CreateKeyVersion(kv *models.KeyVersion) error
	GetKeyVersion(version int64) (*models.KeyVersion, error)
	GetActiveKeyVersion() (*models.KeyVersion, error)
	ListKeyVersions() ([]models.KeyVersion, error)
	UpdateKeyVersion(kv *models.KeyVersion) error
}

// Manager управляет версиями data keys и их статусами.
type Manager struct {
	repo      Repository
	masterKey []byte
	rand      io.Reader
}

// NewManager создаёт менеджер ключей.
func NewManager(repo Repository, masterKey string) (*Manager, error) {
	if repo == nil {
		return nil, errors.New("key repository is required")
	}
	parsed, err := parseMasterKey(masterKey)
	if err != nil {
		return nil, err
	}
	return &Manager{repo: repo, masterKey: parsed, rand: rand.Reader}, nil
}

// EnsureActive возвращает активную версию ключа, создавая её при отсутствии.
func (m *Manager) EnsureActive() (*models.KeyVersion, error) {
	active, err := m.repo.GetActiveKeyVersion()
	if err == nil {
		return active, nil
	}
	if !errors.Is(err, models.ErrUnknownKeyVersion) {
		return nil, err
	}
	return m.createKeyVersion(models.KeyStatusActive)
}

// KeyForEncrypt возвращает data key для шифрования по версии (только active).
func (m *Manager) KeyForEncrypt(version int64) ([]byte, error) {
	kv, err := m.repo.GetKeyVersion(version)
	if err != nil {
		return nil, err
	}
	if !kv.IsActive() {
		return nil, models.ErrKeyVersionNotActive
	}
	return m.unwrapDataKey(kv)
}

// KeyForDecrypt возвращает data key для расшифровки (active/deprecated).
func (m *Manager) KeyForDecrypt(version int64) ([]byte, error) {
	kv, err := m.repo.GetKeyVersion(version)
	if err != nil {
		return nil, err
	}
	if !kv.CanDecrypt() {
		return nil, models.ErrKeyVersionCannotDecrypt
	}
	return m.unwrapDataKey(kv)
}

// CreateActive создаёт новую активную версию ключа.
func (m *Manager) CreateActive() (*models.KeyVersion, error) {
	return m.createKeyVersion(models.KeyStatusActive)
}

// Deprecate переводит активный ключ в deprecated.
func (m *Manager) Deprecate(version int64) error {
	kv, err := m.repo.GetKeyVersion(version)
	if err != nil {
		return err
	}
	if err := kv.Deprecate(); err != nil {
		return err
	}
	return m.repo.UpdateKeyVersion(kv)
}

// Retire переводит deprecated ключ в retired.
func (m *Manager) Retire(version int64) error {
	kv, err := m.repo.GetKeyVersion(version)
	if err != nil {
		return err
	}
	if err := kv.Retire(); err != nil {
		return err
	}
	return m.repo.UpdateKeyVersion(kv)
}

// Rotate депрецирует активный ключ и создаёт новый active.
func (m *Manager) Rotate() (*models.KeyVersion, error) {
	active, err := m.repo.GetActiveKeyVersion()
	if err == nil {
		if err := m.Deprecate(active.Version); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, models.ErrUnknownKeyVersion) {
		return nil, err
	}
	return m.createKeyVersion(models.KeyStatusActive)
}

func (m *Manager) createKeyVersion(status models.KeyStatus) (*models.KeyVersion, error) {
	versions, err := m.repo.ListKeyVersions()
	if err != nil {
		return nil, err
	}
	var maxVersion int64
	for _, v := range versions {
		if v.Version > maxVersion {
			maxVersion = v.Version
		}
	}
	dataKey := make([]byte, dataKeySize)
	if _, err := io.ReadFull(m.rand, dataKey); err != nil {
		return nil, err
	}
	wrapped, nonce, err := m.wrapDataKey(dataKey)
	if err != nil {
		return nil, err
	}
	kv := &models.KeyVersion{
		Version:      maxVersion + 1,
		Status:       status,
		EncryptedKey: wrapped,
		KeyNonce:     nonce,
	}
	if err := m.repo.CreateKeyVersion(kv); err != nil {
		return nil, err
	}
	return kv, nil
}

func (m *Manager) wrapDataKey(dataKey []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, nonceSizeGCM)
	if _, err := io.ReadFull(m.rand, nonce); err != nil {
		return nil, nil, err
	}
	wrapped := gcm.Seal(nil, nonce, dataKey, nil)
	return wrapped, nonce, nil
}

func (m *Manager) unwrapDataKey(kv *models.KeyVersion) ([]byte, error) {
	if kv == nil {
		return nil, errors.New("key version is nil")
	}
	if len(kv.EncryptedKey) == 0 || len(kv.KeyNonce) == 0 {
		return nil, errors.New("key material is missing")
	}
	block, err := aes.NewCipher(m.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	dataKey, err := gcm.Open(nil, kv.KeyNonce, kv.EncryptedKey, nil)
	if err != nil {
		return nil, err
	}
	return dataKey, nil
}

func parseMasterKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, errors.New("master key is empty")
	}
	trimmed := strings.TrimSpace(raw)
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err == nil {
		if len(decoded) != dataKeySize {
			return nil, errors.New("master key must be 32 bytes")
		}
		return decoded, nil
	}
	if len(trimmed) == dataKeySize {
		return []byte(trimmed), nil
	}
	return nil, errors.New("master key must be base64-encoded 32 bytes")
}
