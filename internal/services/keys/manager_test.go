package keys

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

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

func TestCreateActive(t *testing.T) {
	t.Run("creates active key from empty repo", func(t *testing.T) {
		repo := newMemRepo()
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		kv, err := mgr.CreateActive()
		require.NoError(t, err)
		require.NotNil(t, kv)
		require.True(t, kv.IsActive())
		require.Equal(t, int64(1), kv.Version)
		require.NotEmpty(t, kv.EncryptedKey)
		require.NotEmpty(t, kv.KeyNonce)
	})

	t.Run("increments version on subsequent calls", func(t *testing.T) {
		repo := newMemRepo()
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		kv1, err := mgr.CreateActive()
		require.NoError(t, err)
		require.Equal(t, int64(1), kv1.Version)

		kv2, err := mgr.CreateActive()
		require.NoError(t, err)
		require.Equal(t, int64(2), kv2.Version)

		// Both should be active
		require.True(t, kv1.IsActive())
		require.True(t, kv2.IsActive())
	})

	t.Run("created key can be unwrapped and used for encryption", func(t *testing.T) {
		repo := newMemRepo()
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		kv, err := mgr.CreateActive()
		require.NoError(t, err)

		// KeyForEncrypt should succeed for active key
		dataKey, err := mgr.KeyForEncrypt(kv.Version)
		require.NoError(t, err)
		require.Len(t, dataKey, dataKeySize)

		// KeyForDecrypt should also work
		decKey, err := mgr.KeyForDecrypt(kv.Version)
		require.NoError(t, err)
		require.Equal(t, dataKey, decKey)
	})

	t.Run("each CreateActive produces unique data keys", func(t *testing.T) {
		repo := newMemRepo()
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		kv1, err := mgr.CreateActive()
		require.NoError(t, err)
		kv2, err := mgr.CreateActive()
		require.NoError(t, err)

		dk1, err := mgr.KeyForEncrypt(kv1.Version)
		require.NoError(t, err)
		dk2, err := mgr.KeyForEncrypt(kv2.Version)
		require.NoError(t, err)

		require.NotEqual(t, dk1, dk2, "each version should have a unique data key")
	})

	t.Run("persists key in repository", func(t *testing.T) {
		repo := newMemRepo()
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		kv, err := mgr.CreateActive()
		require.NoError(t, err)

		// Verify it was actually stored
		stored, err := repo.GetKeyVersion(kv.Version)
		require.NoError(t, err)
		require.Equal(t, kv.Version, stored.Version)
		require.Equal(t, models.KeyStatusActive, stored.Status)
	})
}

type errRepo struct {
	getActiveErr error
	getErr       error
	listErr      error
	createErr    error
	updateErr    error
	kv           *models.KeyVersion
}

func (e *errRepo) CreateKeyVersion(kv *models.KeyVersion) error { return e.createErr }
func (e *errRepo) GetKeyVersion(version int64) (*models.KeyVersion, error) {
	if e.getErr != nil {
		return nil, e.getErr
	}
	if e.kv != nil {
		return e.kv, nil
	}
	return nil, models.ErrUnknownKeyVersion
}
func (e *errRepo) GetActiveKeyVersion() (*models.KeyVersion, error) {
	if e.getActiveErr != nil {
		return nil, e.getActiveErr
	}
	if e.kv != nil {
		return e.kv, nil
	}
	return nil, models.ErrUnknownKeyVersion
}
func (e *errRepo) ListKeyVersions() ([]models.KeyVersion, error) {
	if e.listErr != nil {
		return nil, e.listErr
	}
	if e.kv != nil {
		return []models.KeyVersion{*e.kv}, nil
	}
	return nil, nil
}
func (e *errRepo) UpdateKeyVersion(kv *models.KeyVersion) error { return e.updateErr }

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestNewManager_Errors(t *testing.T) {
	_, err := NewManager(nil, testMasterKey(t))
	require.EqualError(t, err, "key repository is required")

	_, err = NewManager(newMemRepo(), "")
	require.EqualError(t, err, "master key is empty")
}

func TestParseMasterKey_Formats(t *testing.T) {
	raw := "1234567890123456789012345678901_"
	parsed, err := parseMasterKey(raw)
	require.NoError(t, err)
	require.Equal(t, []byte(raw), parsed)

	base64Key := base64.StdEncoding.EncodeToString([]byte(raw))
	parsed, err = parseMasterKey(base64Key)
	require.NoError(t, err)
	require.Equal(t, []byte(raw), parsed)

	_, err = parseMasterKey(base64.StdEncoding.EncodeToString([]byte("short")))
	require.EqualError(t, err, "master key must be 32 bytes")

	_, err = parseMasterKey("not-base64-and-not-32-bytes")
	require.EqualError(t, err, "master key must be base64-encoded 32 bytes")
}

func TestManager_ErrorBranches(t *testing.T) {
	t.Run("ensure active returns repository error", func(t *testing.T) {
		repo := &errRepo{getActiveErr: errors.New("repo boom")}
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		_, err = mgr.EnsureActive()
		require.EqualError(t, err, "repo boom")
	})

	t.Run("rotate returns repository error", func(t *testing.T) {
		repo := &errRepo{getActiveErr: errors.New("repo boom")}
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		_, err = mgr.Rotate()
		require.EqualError(t, err, "repo boom")
	})

	t.Run("create key version returns list error", func(t *testing.T) {
		repo := &errRepo{listErr: errors.New("list boom")}
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		_, err = mgr.CreateActive()
		require.EqualError(t, err, "list boom")
	})

	t.Run("create key version returns create error", func(t *testing.T) {
		repo := &errRepo{createErr: errors.New("create boom")}
		mgr, err := NewManager(repo, testMasterKey(t))
		require.NoError(t, err)

		_, err = mgr.CreateActive()
		require.EqualError(t, err, "create boom")
	})
}

func TestManager_KeyAccessErrors(t *testing.T) {
	mgr, err := NewManager(newMemRepo(), testMasterKey(t))
	require.NoError(t, err)

	_, err = mgr.unwrapDataKey(nil)
	require.EqualError(t, err, "key version is nil")

	_, err = mgr.unwrapDataKey(&models.KeyVersion{})
	require.EqualError(t, err, "key material is missing")

	repo := &errRepo{kv: &models.KeyVersion{Version: 1, Status: models.KeyStatusDeprecated}}
	mgr, err = NewManager(repo, testMasterKey(t))
	require.NoError(t, err)
	_, err = mgr.KeyForEncrypt(1)
	require.ErrorIs(t, err, models.ErrKeyVersionNotActive)

	repo = &errRepo{kv: &models.KeyVersion{Version: 1, Status: models.KeyStatusRetired}}
	mgr, err = NewManager(repo, testMasterKey(t))
	require.NoError(t, err)
	_, err = mgr.KeyForDecrypt(1)
	require.ErrorIs(t, err, models.ErrKeyVersionCannotDecrypt)
}

func TestManager_RandFailures(t *testing.T) {
	mgr, err := NewManager(newMemRepo(), testMasterKey(t))
	require.NoError(t, err)

	mgr.rand = errReader{}
	_, err = mgr.CreateActive()
	require.Error(t, err)
}
