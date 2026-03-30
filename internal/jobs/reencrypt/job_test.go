package reencrypt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

// ---------------------------------------------------------------------------
// Simple struct-based mocks
// ---------------------------------------------------------------------------

// mockRepo implements Repository for unit tests.
type mockRepo struct {
	records             []models.Record
	payloads            map[int64][]models.StoredPayload
	updatedRecords      []*models.Record
	updatedPayloadSizes []struct{ recordID, version, size int64 }
	listErr             error
	updateErr           error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		payloads: make(map[int64][]models.StoredPayload),
	}
}

func (m *mockRepo) ListRecordsForReencrypt(activeVersion int64, limit int) ([]models.Record, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []models.Record
	for _, r := range m.records {
		if r.KeyVersion != activeVersion {
			result = append(result, r)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockRepo) UpdateRecord(record *models.Record) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	recordCopy := *record
	m.updatedRecords = append(m.updatedRecords, &recordCopy)
	for i := range m.records {
		if m.records[i].ID == record.ID {
			m.records[i] = recordCopy
			break
		}
	}
	return nil
}

func (m *mockRepo) ListPayloads(recordID int64) ([]models.StoredPayload, error) {
	return m.payloads[recordID], nil
}

func (m *mockRepo) UpdatePayloadSize(recordID int64, version int64, size int64) error {
	m.updatedPayloadSizes = append(m.updatedPayloadSizes, struct{ recordID, version, size int64 }{recordID, version, size})
	return nil
}

// mockBlob implements repositories.BlobStorage.
type mockBlob struct {
	data map[string][]byte
	err  map[string]error
}

func newMockBlob() *mockBlob {
	return &mockBlob{
		data: make(map[string][]byte),
		err:  make(map[string]error),
	}
}

func (m *mockBlob) Save(path string, data []byte) error {
	if err, ok := m.err[path]; ok {
		return err
	}
	m.data[path] = data
	return nil
}

func (m *mockBlob) Read(path string) ([]byte, error) {
	if err, ok := m.err[path]; ok {
		return nil, err
	}
	d, ok := m.data[path]
	if !ok {
		return nil, fmt.Errorf("blob not found: %s", path)
	}
	return d, nil
}

func (m *mockBlob) Delete(path string) error {
	delete(m.data, path)
	return nil
}

func (m *mockBlob) Exists(path string) (bool, error) {
	_, ok := m.data[path]
	return ok, nil
}

// mockCrypto implements cryptosvc.CryptoService with a simple XOR for testing.
type mockCrypto struct{}

func newMockCrypto() *mockCrypto {
	return &mockCrypto{}
}

func (m *mockCrypto) Encrypt(data []byte, keyVersion int64) ([]byte, error) {
	// prepend "GK1" prefix + 12-byte zero nonce + XORed data
	result := make([]byte, 0, 3+12+len(data))
	result = append(result, "GK1"...)
	result = append(result, make([]byte, 12)...)
	for _, b := range data {
		result = append(result, b^0xFF)
	}
	return result, nil
}

func (m *mockCrypto) Decrypt(data []byte, keyVersion int64) ([]byte, error) {
	if len(data) < 3+12 {
		return nil, fmt.Errorf("too short")
	}
	if string(data[:3]) != "GK1" {
		return nil, fmt.Errorf("bad prefix")
	}
	plain := make([]byte, 0, len(data)-15)
	for _, b := range data[15:] {
		plain = append(plain, b^0xFF)
	}
	return plain, nil
}

// ---------------------------------------------------------------------------
// Minimal in-memory key repo for building a real keys.Manager
// ---------------------------------------------------------------------------

type memKeyRepo struct {
	versions map[int64]*models.KeyVersion
}

func newMemKeyRepo() *memKeyRepo {
	return &memKeyRepo{versions: make(map[int64]*models.KeyVersion)}
}

func (m *memKeyRepo) CreateKeyVersion(kv *models.KeyVersion) error {
	c := *kv
	m.versions[kv.Version] = &c
	return nil
}

func (m *memKeyRepo) GetKeyVersion(version int64) (*models.KeyVersion, error) {
	kv, ok := m.versions[version]
	if !ok {
		return nil, models.ErrUnknownKeyVersion
	}
	c := *kv
	return &c, nil
}

func (m *memKeyRepo) GetActiveKeyVersion() (*models.KeyVersion, error) {
	var active *models.KeyVersion
	for _, kv := range m.versions {
		if kv.Status == models.KeyStatusActive {
			if active == nil || kv.Version > active.Version {
				c := *kv
				active = &c
			}
		}
	}
	if active == nil {
		return nil, models.ErrUnknownKeyVersion
	}
	return active, nil
}

func (m *memKeyRepo) ListKeyVersions() ([]models.KeyVersion, error) {
	result := make([]models.KeyVersion, 0, len(m.versions))
	for _, kv := range m.versions {
		result = append(result, *kv)
	}
	return result, nil
}

func (m *memKeyRepo) UpdateKeyVersion(kv *models.KeyVersion) error {
	stored, ok := m.versions[kv.Version]
	if !ok {
		return models.ErrUnknownKeyVersion
	}
	stored.Status = kv.Status
	stored.DeprecatedAt = kv.DeprecatedAt
	stored.RetiredAt = kv.RetiredAt
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testJobMasterKeyLocal(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(key)
}

func setupKeysManager(t *testing.T) *keys.Manager {
	t.Helper()
	repo := newMemKeyRepo()
	mgr, err := keys.NewManager(repo, testJobMasterKeyLocal(t))
	require.NoError(t, err)
	return mgr
}

// buildRealCrypto creates a real crypto.Service backed by a keys.Manager.
// This is needed because runOnce/reencryptBinary call the real crypto logic.
func buildRealCrypto(t *testing.T) (*keys.Manager, cryptosvc.CryptoService) {
	t.Helper()
	mgr := setupKeysManager(t)
	svc := cryptosvc.New(mgr)
	return mgr, svc
}

// ---------------------------------------------------------------------------
// Tests: New, With*, Start, Stop
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		job := New()
		require.NotNil(t, job)
		require.Equal(t, defaultBatchSize, job.batchSize)
		require.Equal(t, defaultInterval, job.interval)
		require.False(t, job.enabled)
		require.NotNil(t, job.stopCh)
		require.NotNil(t, job.doneCh)
	})
}

func TestWithDeps(t *testing.T) {
	t.Run("sets all deps and enables job", func(t *testing.T) {
		repo := newMockRepo()
		blob := newMockBlob()
		crypto := newMockCrypto()
		mgr := setupKeysManager(t)

		job := New(WithDeps(repo, blob, crypto, mgr))
		require.True(t, job.enabled)
		require.Equal(t, repo, job.repo)
		require.Equal(t, blob, job.blob)
	})
}

func TestWithBatchSize(t *testing.T) {
	t.Run("overrides default batch size", func(t *testing.T) {
		job := New(WithBatchSize(42))
		require.Equal(t, 42, job.batchSize)
	})
}

func TestWithInterval(t *testing.T) {
	t.Run("overrides default interval", func(t *testing.T) {
		custom := 5 * time.Minute
		job := New(WithInterval(custom))
		require.Equal(t, custom, job.interval)
	})
}

func TestStartWithoutDeps(t *testing.T) {
	t.Run("Start returns nil when not enabled", func(t *testing.T) {
		job := New()
		err := job.Start(context.Background())
		require.NoError(t, err)
	})
}

func TestStopWithoutDeps(t *testing.T) {
	t.Run("Stop returns nil when not enabled", func(t *testing.T) {
		job := New()
		err := job.Stop(context.Background())
		require.NoError(t, err)
	})
}

func TestStopWithTimeout(t *testing.T) {
	t.Run("Stop returns context error on cancelled context", func(t *testing.T) {
		repo := newMockRepo()
		blob := newMockBlob()
		crypto := newMockCrypto()
		mgr := setupKeysManager(t)

		job := New(WithDeps(repo, blob, crypto, mgr))

		ctx := context.Background()
		_ = job.Start(ctx)

		// Use an already-cancelled context for Stop
		stopCtx, stopCancel := context.WithCancel(context.Background())
		stopCancel()

		err := job.Stop(stopCtx)
		require.Error(t, err)
	})
}

func TestStopAfterStart(t *testing.T) {
	t.Run("Stop completes normally after Start", func(t *testing.T) {
		repo := newMockRepo()
		blob := newMockBlob()
		crypto := newMockCrypto()
		mgr := setupKeysManager(t)

		job := New(WithDeps(repo, blob, crypto, mgr))

		ctx := context.Background()
		_ = job.Start(ctx)

		err := job.Stop(ctx)
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// Tests: runOnce
// ---------------------------------------------------------------------------

func TestRunOnceNoRecords(t *testing.T) {
	t.Run("returns nil when no records need reencryption", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		repo := newMockRepo()
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))
		err := job.runOnce(context.Background())
		require.NoError(t, err)
	})
}

func TestRunOnceCancelledContext(t *testing.T) {
	t.Run("returns context error", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		repo := newMockRepo()
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := job.runOnce(ctx)
		require.Error(t, err)
	})
}

func TestRunOnceListError(t *testing.T) {
	t.Run("propagates ListRecordsForReencrypt error", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		repo := newMockRepo()
		repo.listErr = fmt.Errorf("db down")
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))
		err := job.runOnce(context.Background())
		require.EqualError(t, err, "db down")
	})
}

func TestRunOnceUpdateError(t *testing.T) {
	t.Run("propagates UpdateRecord error", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		active, err := mgr.EnsureActive()
		require.NoError(t, err)

		repo := newMockRepo()
		repo.records = []models.Record{
			{ID: 1, Type: models.RecordTypeText, KeyVersion: active.Version - 1},
		}
		repo.updateErr = fmt.Errorf("update failed")
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))
		err = job.runOnce(context.Background())
		require.EqualError(t, err, "update failed")
	})
}

func TestRunOnceSkipsAlreadyActive(t *testing.T) {
	t.Run("skips records already on active version", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		active, err := mgr.EnsureActive()
		require.NoError(t, err)

		repo := newMockRepo()
		// Record with same key version as active - should be skipped by ListRecordsForReencrypt
		repo.records = []models.Record{
			{ID: 1, Type: models.RecordTypeText, KeyVersion: active.Version},
		}
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))
		err = job.runOnce(context.Background())
		require.NoError(t, err)
		require.Empty(t, repo.updatedRecords)
	})
}

// ---------------------------------------------------------------------------
// Tests: reencryptBinary
// ---------------------------------------------------------------------------

func TestReencryptBinary(t *testing.T) {
	t.Run("reencrypts binary payload and updates blob", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		active, err := mgr.EnsureActive()
		require.NoError(t, err)

		// Create a separate old version to simulate old key
		oldActive, err := mgr.CreateActive()
		require.NoError(t, err)

		// Encrypt original data with old version
		originalData := []byte("secret binary content")
		encryptedData, err := crypto.Encrypt(originalData, oldActive.Version)
		require.NoError(t, err)

		blob := newMockBlob()
		blob.data["blobs/1/v1"] = encryptedData

		repo := newMockRepo()
		repo.payloads[1] = []models.StoredPayload{
			{RecordID: 1, Version: 1, StoragePath: "blobs/1/v1", Size: int64(len(encryptedData))},
		}

		job := New(WithDeps(repo, blob, crypto, mgr))

		record := &models.Record{
			ID:         1,
			Type:       models.RecordTypeBinary,
			KeyVersion: oldActive.Version,
		}

		err = job.reencryptBinary(record, active.Version)
		require.NoError(t, err)

		// Blob should have been overwritten
		newData, ok := blob.data["blobs/1/v1"]
		require.True(t, ok)
		require.NotEqual(t, encryptedData, newData)

		// Payload size should have been updated
		require.Len(t, repo.updatedPayloadSizes, 1)
		require.Equal(t, int64(1), repo.updatedPayloadSizes[0].recordID)

		// Verify we can decrypt with new version
		decrypted, err := crypto.Decrypt(newData, active.Version)
		require.NoError(t, err)
		require.Equal(t, originalData, decrypted)
	})

	t.Run("returns error when blob read fails", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		active, err := mgr.EnsureActive()
		require.NoError(t, err)

		blob := newMockBlob()
		blob.err["missing"] = fmt.Errorf("read error")

		repo := newMockRepo()
		repo.payloads[2] = []models.StoredPayload{
			{RecordID: 2, Version: 1, StoragePath: "missing", Size: 100},
		}

		job := New(WithDeps(repo, blob, crypto, mgr))

		record := &models.Record{
			ID:         2,
			Type:       models.RecordTypeBinary,
			KeyVersion: 0,
		}

		err = job.reencryptBinary(record, active.Version)
		require.EqualError(t, err, "read error")
	})
}

// ---------------------------------------------------------------------------
// Tests: decryptMaybeLegacy
// ---------------------------------------------------------------------------

func TestDecryptMaybeLegacy(t *testing.T) {
	t.Run("empty data returned as-is", func(t *testing.T) {
		crypto := newMockCrypto()

		result, err := decryptMaybeLegacy(crypto, nil, 1)
		require.NoError(t, err)
		require.Nil(t, result)

		result, err = decryptMaybeLegacy(crypto, []byte{}, 1)
		require.NoError(t, err)
		require.Empty(t, result)
	})

	t.Run("data without GK1 prefix returned as-is (legacy)", func(t *testing.T) {
		crypto := newMockCrypto()
		data := []byte("plain text data")

		result, err := decryptMaybeLegacy(crypto, data, 1)
		require.NoError(t, err)
		require.Equal(t, data, result)
	})

	t.Run("data with GK1 prefix is decrypted", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		active, err := mgr.EnsureActive()
		require.NoError(t, err)

		original := []byte("encrypted payload")
		encrypted, err := crypto.Encrypt(original, active.Version)
		require.NoError(t, err)

		result, err := decryptMaybeLegacy(crypto, encrypted, active.Version)
		require.NoError(t, err)
		require.Equal(t, original, result)
	})
}

// ---------------------------------------------------------------------------
// Tests: runOnce end-to-end with text records
// ---------------------------------------------------------------------------

func TestRunOnceTextRecord(t *testing.T) {
	t.Run("updates key version for text records", func(t *testing.T) {
		mgr, crypto := buildRealCrypto(t)
		oldActive, err := mgr.CreateActive()
		require.NoError(t, err)

		// Rotate to get a new active version
		newActive, err := mgr.Rotate()
		require.NoError(t, err)
		require.NotEqual(t, oldActive.Version, newActive.Version)

		repo := newMockRepo()
		repo.records = []models.Record{
			{ID: 10, Type: models.RecordTypeText, KeyVersion: oldActive.Version},
			{ID: 11, Type: models.RecordTypeText, KeyVersion: oldActive.Version},
		}
		blob := newMockBlob()

		job := New(WithDeps(repo, blob, crypto, mgr))
		err = job.runOnce(context.Background())
		require.NoError(t, err)

		// Both records should have been updated
		require.Len(t, repo.updatedRecords, 2)
		for _, r := range repo.updatedRecords {
			require.Equal(t, newActive.Version, r.KeyVersion)
		}
	})
}
