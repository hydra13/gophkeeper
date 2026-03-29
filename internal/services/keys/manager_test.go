package keys

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
)

type memRepo struct {
	versions map[int64]*models.KeyVersion
}

func newMemRepo() *memRepo {
	return &memRepo{versions: make(map[int64]*models.KeyVersion)}
}

func (m *memRepo) CreateKeyVersion(kv *models.KeyVersion) error {
	copy := *kv
	m.versions[kv.Version] = &copy
	return nil
}

func (m *memRepo) GetKeyVersion(version int64) (*models.KeyVersion, error) {
	kv, ok := m.versions[version]
	if !ok {
		return nil, models.ErrUnknownKeyVersion
	}
	copy := *kv
	return &copy, nil
}

func (m *memRepo) GetActiveKeyVersion() (*models.KeyVersion, error) {
	var active *models.KeyVersion
	for _, kv := range m.versions {
		if kv.Status == models.KeyStatusActive {
			if active == nil || kv.Version > active.Version {
				copy := *kv
				active = &copy
			}
		}
	}
	if active == nil {
		return nil, models.ErrUnknownKeyVersion
	}
	return active, nil
}

func (m *memRepo) ListKeyVersions() ([]models.KeyVersion, error) {
	result := make([]models.KeyVersion, 0, len(m.versions))
	for _, kv := range m.versions {
		result = append(result, *kv)
	}
	return result, nil
}

func (m *memRepo) UpdateKeyVersion(kv *models.KeyVersion) error {
	stored, ok := m.versions[kv.Version]
	if !ok {
		return models.ErrUnknownKeyVersion
	}
	stored.Status = kv.Status
	stored.DeprecatedAt = kv.DeprecatedAt
	stored.RetiredAt = kv.RetiredAt
	return nil
}

func TestKeyLifecycle(t *testing.T) {
	repo := newMemRepo()
	master := testMasterKey(t)
	manager, err := NewManager(repo, master)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	active, err := manager.EnsureActive()
	if err != nil {
		t.Fatalf("ensure active error: %v", err)
	}
	if !active.IsActive() {
		t.Fatal("expected active key")
	}
	if len(active.EncryptedKey) == 0 || len(active.KeyNonce) == 0 {
		t.Fatal("expected key material")
	}

	_, err = manager.KeyForEncrypt(active.Version)
	if err != nil {
		t.Fatalf("key for encrypt error: %v", err)
	}

	_, err = manager.Rotate()
	if err != nil {
		t.Fatalf("rotate error: %v", err)
	}

	old, err := repo.GetKeyVersion(active.Version)
	if err != nil {
		t.Fatalf("old version fetch error: %v", err)
	}
	if old.Status != models.KeyStatusDeprecated {
		t.Fatalf("expected deprecated, got %s", old.Status)
	}

	if _, err := manager.KeyForEncrypt(old.Version); err == nil {
		t.Fatal("expected error for encrypt with deprecated key")
	}
	if _, err := manager.KeyForDecrypt(old.Version); err != nil {
		t.Fatalf("expected decrypt for deprecated key, got %v", err)
	}

	if err := manager.Retire(old.Version); err != nil {
		t.Fatalf("retire error: %v", err)
	}
	if _, err := manager.KeyForDecrypt(old.Version); err == nil {
		t.Fatal("expected error for decrypt with retired key")
	}
}

func testMasterKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("rand error: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}
