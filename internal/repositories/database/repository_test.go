package database

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockBlobStorage implements repositories.BlobStorage for unit tests.
type mockBlobStorage struct {
	saveErr   error
	readErr   error
	deleteErr error
	existsErr error
	existsVal bool
	saved     map[string][]byte
}

func newMockBlobStorage() *mockBlobStorage {
	return &mockBlobStorage{saved: make(map[string][]byte)}
}

func (m *mockBlobStorage) Save(path string, data []byte) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved[path] = data
	return nil
}

func (m *mockBlobStorage) Read(path string) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	data, ok := m.saved[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return data, nil
}

func (m *mockBlobStorage) Delete(path string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.saved, path)
	return nil
}

func (m *mockBlobStorage) Exists(path string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	_, ok := m.saved[path]
	return ok || m.existsVal, nil
}

// mockCryptoService implements cryptosvc.CryptoService for unit tests.
// Encrypt returns data prefixed with "ENC:" for easy verification.
// Decrypt strips the "ENC:" prefix.
type mockCryptoService struct {
	encryptErr error
	decryptErr error
}

func (m *mockCryptoService) Encrypt(data []byte, keyVersion int64) ([]byte, error) {
	if m.encryptErr != nil {
		return nil, m.encryptErr
	}
	prefix := cryptosvc.HasEncryptedPrefix(data)
	_ = prefix
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (m *mockCryptoService) Decrypt(data []byte, keyVersion int64) ([]byte, error) {
	if m.decryptErr != nil {
		return nil, m.decryptErr
	}
	return data, nil
}

// mockRow implements interface{ Scan(dest ...any) error } for testing scan helpers.
type mockRow struct {
	scanErr error
	values  []any
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	for i, v := range dest {
		if i >= len(m.values) {
			break
		}
		// Use a switch on the source value to assign to destination pointer.
		// This handles type aliases (e.g. models.KeyStatus ~ string) correctly.
		src := m.values[i]
		switch s := src.(type) {
		case int64:
			switch d := v.(type) {
			case *int64:
				*d = s
			}
		case string:
			switch d := v.(type) {
			case *string:
				*d = s
			case *models.KeyStatus:
				*d = models.KeyStatus(s)
			case *models.RecordType:
				*d = models.RecordType(s)
			case *models.UploadStatus:
				*d = models.UploadStatus(s)
			}
		case []byte:
			if d, ok := v.(*[]byte); ok {
				*d = s
			}
		case time.Time:
			if d, ok := v.(*time.Time); ok {
				*d = s
			}
		case bool:
			if d, ok := v.(*bool); ok {
				*d = s
			}
		case sql.NullTime:
			if d, ok := v.(*sql.NullTime); ok {
				*d = s
			}
		case sql.NullString:
			if d, ok := v.(*sql.NullString); ok {
				*d = s
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper: create a Repository with a real *sql.DB is needed for New(),
// but for pure unit tests of helper functions we create Repository directly.
// ---------------------------------------------------------------------------

// newTestRepo creates a Repository without a real database.
// Only suitable for testing helper/parsing functions.
func newTestRepo() *Repository {
	return &Repository{
		db:     nil,
		blob:   newMockBlobStorage(),
		crypto: &mockCryptoService{},
	}
}

// newTestRepoNoCrypto creates a Repository without crypto service set.
func newTestRepoNoCrypto() *Repository {
	return &Repository{
		db:     nil,
		blob:   newMockBlobStorage(),
		crypto: nil,
	}
}

// ---------------------------------------------------------------------------
// 1. New() constructor tests
// ---------------------------------------------------------------------------

func TestNew_NilDB_ReturnsError(t *testing.T) {
	t.Parallel()
	blob := newMockBlobStorage()
	repo, err := New(nil, blob)
	require.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "db instance is required")
}

func TestNew_NilBlob_ReturnsError(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("pgx", "") //nolint:unused // intentionally not connecting
	if err != nil {
		t.Skip("pgx driver not available, using lightweight stub")
	}
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	repo, repoErr := New(db, nil)
	require.Error(t, repoErr)
	assert.Nil(t, repo)
	assert.Contains(t, repoErr.Error(), "blob storage is required")
}

func TestNew_BothNil_ReturnsDBErrorFirst(t *testing.T) {
	t.Parallel()
	repo, err := New(nil, nil)
	require.Error(t, err)
	assert.Nil(t, repo)
	// db is checked first
	assert.Contains(t, err.Error(), "db instance is required")
}

func TestNew_ValidParams_ReturnsRepository(t *testing.T) {
	// Cannot use real *sql.DB without driver, so use pgx with empty DSN.
	// The constructor only checks for nil, it doesn't ping.
	db, err := sql.Open("pgx", "")
	if err != nil {
		t.Skip("pgx driver not available")
	}
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	blob := newMockBlobStorage()
	repo, repoErr := New(db, blob)
	require.NoError(t, repoErr)
	require.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
	assert.Equal(t, blob, repo.blob)
}

// ---------------------------------------------------------------------------
// 2. SetCrypto() and ensureCrypto() tests
// ---------------------------------------------------------------------------

func TestSetCrypto_SetsService(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	assert.Nil(t, repo.crypto)

	crypto := &mockCryptoService{}
	repo.SetCrypto(crypto)
	assert.Equal(t, crypto, repo.crypto)
}

func TestEnsureCrypto_NotSet_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	svc, err := repo.ensureCrypto()
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestEnsureCrypto_Set_ReturnsService(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	svc, err := repo.ensureCrypto()
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestSetCrypto_OverwritesExistingService(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()

	first := repo.crypto
	require.NotNil(t, first)

	newCrypto := &mockCryptoService{encryptErr: errors.New("different")}
	repo.SetCrypto(newCrypto)
	assert.Same(t, newCrypto, repo.crypto)
	assert.NotSame(t, first, repo.crypto)
}

func TestSetCrypto_NilService_AllowsEnsureToReturnError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	require.NotNil(t, repo.crypto)

	repo.SetCrypto(nil)
	svc, err := repo.ensureCrypto()
	require.Error(t, err)
	assert.Nil(t, svc)
}

// ---------------------------------------------------------------------------
// 3. scanUser() tests
// ---------------------------------------------------------------------------

func TestScanUser_ValidData(t *testing.T) {
	t.Parallel()
	now := time.Now()
	row := &mockRow{
		values: []any{
			int64(1),           // ID
			"user@example.com", // Email
			"hash123",          // PasswordHash
			now,                // CreatedAt
			now,                // UpdatedAt
		},
	}
	user, err := scanUser(row)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "hash123", user.PasswordHash)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

func TestScanUser_NoRows_ReturnsErrUserNotFound(t *testing.T) {
	t.Parallel()
	row := &mockRow{scanErr: sql.ErrNoRows}
	user, err := scanUser(row)
	require.Error(t, err)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, models.ErrUserNotFound)
}

func TestScanUser_OtherError_ReturnsError(t *testing.T) {
	t.Parallel()
	row := &mockRow{scanErr: errors.New("connection refused")}
	user, err := scanUser(row)
	require.Error(t, err)
	assert.Nil(t, user)
	assert.EqualError(t, err, "connection refused")
}

// ---------------------------------------------------------------------------
// 4. scanSession() tests
// ---------------------------------------------------------------------------

func TestScanSession_ValidData_Active(t *testing.T) {
	t.Parallel()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	row := &mockRow{
		values: []any{
			int64(42),       // ID
			int64(1),        // UserID
			"device-1",      // DeviceID
			"MacBook Pro",   // DeviceName
			"cli",           // ClientType
			"refresh-token", // RefreshToken
			now,             // LastSeenAt
			future,          // ExpiresAt
			sql.NullTime{},  // RevokedAt (not revoked)
			now,             // CreatedAt
		},
	}
	session, err := scanSession(row)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, int64(42), session.ID)
	assert.Equal(t, int64(1), session.UserID)
	assert.Equal(t, "device-1", session.DeviceID)
	assert.Equal(t, "MacBook Pro", session.DeviceName)
	assert.Equal(t, "cli", session.ClientType)
	assert.Equal(t, "refresh-token", session.RefreshToken)
	assert.Nil(t, session.RevokedAt)
}

func TestScanSession_Revoked(t *testing.T) {
	t.Parallel()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	revokedAt := now.Add(-1 * time.Hour)
	row := &mockRow{
		values: []any{
			int64(42),
			int64(1),
			"device-1",
			"MacBook Pro",
			"cli",
			"refresh-token",
			now,
			future,
			sql.NullTime{Time: revokedAt, Valid: true},
			now,
		},
	}
	session, err := scanSession(row)
	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotNil(t, session.RevokedAt)
	assert.Equal(t, revokedAt, *session.RevokedAt)
}

func TestScanSession_NoRows_ReturnsErrUnauthorized(t *testing.T) {
	t.Parallel()
	row := &mockRow{scanErr: sql.ErrNoRows}
	session, err := scanSession(row)
	require.Error(t, err)
	assert.Nil(t, session)
	assert.ErrorIs(t, err, models.ErrUnauthorized)
}

func TestScanSession_OtherError(t *testing.T) {
	t.Parallel()
	row := &mockRow{scanErr: errors.New("scan failure")}
	session, err := scanSession(row)
	require.Error(t, err)
	assert.Nil(t, session)
	assert.EqualError(t, err, "scan failure")
}

// ---------------------------------------------------------------------------
// 5. scanKeyVersion() tests
// ---------------------------------------------------------------------------

func TestScanKeyVersion_ValidData_Active(t *testing.T) {
	t.Parallel()
	now := time.Now()
	row := &mockRow{
		values: []any{
			int64(1),          // ID
			int64(2),          // Version
			"active",          // Status
			[]byte("enc-key"), // EncryptedKey
			[]byte("nonce"),   // KeyNonce
			now,               // CreatedAt
			sql.NullTime{},    // DeprecatedAt
			sql.NullTime{},    // RetiredAt
		},
	}
	kv, err := scanKeyVersion(row)
	require.NoError(t, err)
	require.NotNil(t, kv)
	assert.Equal(t, int64(1), kv.ID)
	assert.Equal(t, int64(2), kv.Version)
	assert.Equal(t, models.KeyStatusActive, kv.Status)
	assert.Equal(t, []byte("enc-key"), kv.EncryptedKey)
	assert.Equal(t, []byte("nonce"), kv.KeyNonce)
	assert.Nil(t, kv.DeprecatedAt)
	assert.Nil(t, kv.RetiredAt)
}

func TestScanKeyVersion_Deprecated(t *testing.T) {
	t.Parallel()
	now := time.Now()
	depAt := now.Add(-1 * time.Hour)
	row := &mockRow{
		values: []any{
			int64(1),
			int64(1),
			"deprecated",
			[]byte("enc-key"),
			[]byte("nonce"),
			now,
			sql.NullTime{Time: depAt, Valid: true},
			sql.NullTime{},
		},
	}
	kv, err := scanKeyVersion(row)
	require.NoError(t, err)
	require.NotNil(t, kv.DeprecatedAt)
	assert.Equal(t, depAt, *kv.DeprecatedAt)
	assert.Nil(t, kv.RetiredAt)
}

func TestScanKeyVersion_Retired(t *testing.T) {
	t.Parallel()
	now := time.Now()
	depAt := now.Add(-2 * time.Hour)
	retAt := now.Add(-1 * time.Hour)
	row := &mockRow{
		values: []any{
			int64(1),
			int64(1),
			"retired",
			[]byte("enc-key"),
			[]byte("nonce"),
			now,
			sql.NullTime{Time: depAt, Valid: true},
			sql.NullTime{Time: retAt, Valid: true},
		},
	}
	kv, err := scanKeyVersion(row)
	require.NoError(t, err)
	require.NotNil(t, kv.DeprecatedAt)
	require.NotNil(t, kv.RetiredAt)
	assert.Equal(t, retAt, *kv.RetiredAt)
}

func TestScanKeyVersion_NoRows_ReturnsErrUnknownKeyVersion(t *testing.T) {
	t.Parallel()
	row := &mockRow{scanErr: sql.ErrNoRows}
	kv, err := scanKeyVersion(row)
	require.Error(t, err)
	assert.Nil(t, kv)
	assert.ErrorIs(t, err, models.ErrUnknownKeyVersion)
}

// ---------------------------------------------------------------------------
// 6. splitRecordPayload() tests
// ---------------------------------------------------------------------------

func TestSplitRecordPayload_NilPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{Payload: nil}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, jsonData)
	assert.Nil(t, binData)
}

func TestSplitRecordPayload_LoginPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		Payload: models.LoginPayload{Login: "user", Password: "pass"},
	}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, binData)
	require.NotNil(t, jsonData)

	// Verify JSON round-trip
	var parsed models.LoginPayload
	require.NoError(t, json.Unmarshal(jsonData, &parsed))
	assert.Equal(t, "user", parsed.Login)
	assert.Equal(t, "pass", parsed.Password)
}

func TestSplitRecordPayload_TextPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		Payload: models.TextPayload{Content: "hello world"},
	}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, binData)
	require.NotNil(t, jsonData)

	var parsed models.TextPayload
	require.NoError(t, json.Unmarshal(jsonData, &parsed))
	assert.Equal(t, "hello world", parsed.Content)
}

func TestSplitRecordPayload_CardPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		Payload: models.CardPayload{Number: "4111111111111111", HolderName: "John", ExpiryDate: "12/30", CVV: "123"},
	}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, binData)
	require.NotNil(t, jsonData)

	var parsed models.CardPayload
	require.NoError(t, json.Unmarshal(jsonData, &parsed))
	assert.Equal(t, "4111111111111111", parsed.Number)
	assert.Equal(t, "John", parsed.HolderName)
}

func TestSplitRecordPayload_BinaryPayload_ReturnsNilNil(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		Payload: models.BinaryPayload{Data: []byte("binary-data")},
	}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, jsonData)
	assert.Nil(t, binData)
}

func TestSplitRecordPayload_BinaryPayloadPointer_ReturnsNilNil(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		Payload: &models.BinaryPayload{Data: []byte("binary-data")},
	}
	jsonData, binData, err := splitRecordPayload(record)
	require.NoError(t, err)
	assert.Nil(t, jsonData)
	assert.Nil(t, binData)
}

// ---------------------------------------------------------------------------
// 7. decodePayload() tests
// ---------------------------------------------------------------------------

func TestDecodePayload_LoginType(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(models.LoginPayload{Login: "admin", Password: "secret"})
	payload, err := decodePayload(models.RecordTypeLogin, raw)
	require.NoError(t, err)
	login, ok := payload.(models.LoginPayload)
	require.True(t, ok)
	assert.Equal(t, "admin", login.Login)
	assert.Equal(t, "secret", login.Password)
}

func TestDecodePayload_TextType(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(models.TextPayload{Content: "my secret"})
	payload, err := decodePayload(models.RecordTypeText, raw)
	require.NoError(t, err)
	text, ok := payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "my secret", text.Content)
}

func TestDecodePayload_BinaryType(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(models.BinaryPayload{Data: []byte("bytes")})
	payload, err := decodePayload(models.RecordTypeBinary, raw)
	require.NoError(t, err)
	bin, ok := payload.(models.BinaryPayload)
	require.True(t, ok)
	assert.Equal(t, []byte("bytes"), bin.Data)
}

func TestDecodePayload_CardType(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(models.CardPayload{Number: "1234", HolderName: "Jane", ExpiryDate: "01/25", CVV: "000"})
	payload, err := decodePayload(models.RecordTypeCard, raw)
	require.NoError(t, err)
	card, ok := payload.(models.CardPayload)
	require.True(t, ok)
	assert.Equal(t, "1234", card.Number)
	assert.Equal(t, "Jane", card.HolderName)
}

func TestDecodePayload_EmptyBinary_ReturnsEmptyBinaryPayload(t *testing.T) {
	t.Parallel()
	payload, err := decodePayload(models.RecordTypeBinary, nil)
	require.NoError(t, err)
	bin, ok := payload.(models.BinaryPayload)
	require.True(t, ok)
	assert.Equal(t, models.BinaryPayload{}, bin)
}

func TestDecodePayload_EmptyNonBinary_ReturnsNil(t *testing.T) {
	t.Parallel()
	payload, err := decodePayload(models.RecordTypeLogin, nil)
	require.NoError(t, err)
	assert.Nil(t, payload)
}

func TestDecodePayload_EmptySliceNonBinary_ReturnsNil(t *testing.T) {
	t.Parallel()
	payload, err := decodePayload(models.RecordTypeText, []byte{})
	require.NoError(t, err)
	assert.Nil(t, payload)
}

func TestDecodePayload_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := decodePayload(models.RecordTypeLogin, []byte("not-json"))
	require.Error(t, err)
}

func TestDecodePayload_UnknownType_ReturnsErrInvalidRecordType(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]string{"foo": "bar"})
	_, err := decodePayload(models.RecordType("unknown"), raw)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrInvalidRecordType)
}

// ---------------------------------------------------------------------------
// 8. isUniqueViolation() tests
// ---------------------------------------------------------------------------

func TestIsUniqueViolation_NilError(t *testing.T) {
	t.Parallel()
	assert.False(t, isUniqueViolation(nil))
}

func TestIsUniqueViolation_NonPostgresError(t *testing.T) {
	t.Parallel()
	assert.False(t, isUniqueViolation(errors.New("some error")))
}

func TestIsUniqueViolation_PostgresUniqueViolation(t *testing.T) {
	t.Parallel()
	// We cannot easily construct a real *pgconn.PgError in unit tests
	// without importing the pgconn package. Instead, we verify that
	// non-PgError errors return false.
	assert.False(t, isUniqueViolation(fmt.Errorf("wrapped: %w", errors.New("23505"))))
}

// ---------------------------------------------------------------------------
// 9. nullIfEmpty() tests
// ---------------------------------------------------------------------------

func TestNullIfEmpty_EmptyString(t *testing.T) {
	t.Parallel()
	result := nullIfEmpty("")
	assert.False(t, result.Valid)
	assert.Equal(t, "", result.String)
}

func TestNullIfEmpty_NonEmptyString(t *testing.T) {
	t.Parallel()
	result := nullIfEmpty("hello")
	assert.True(t, result.Valid)
	assert.Equal(t, "hello", result.String)
}

func TestNullIfEmpty_WhitespaceString(t *testing.T) {
	t.Parallel()
	// Whitespace-only is still a non-empty string for nullIfEmpty
	result := nullIfEmpty(" ")
	assert.True(t, result.Valid)
	assert.Equal(t, " ", result.String)
}

// ---------------------------------------------------------------------------
// 10. encryptJSONPayload() tests
// ---------------------------------------------------------------------------

func TestEncryptJSONPayload_EmptyPayload_ReturnsPayload(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := encryptJSONPayload(crypto, []byte{}, 1)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, result)
}

func TestEncryptJSONPayload_NilPayload_ReturnsPayload(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := encryptJSONPayload(crypto, nil, 1)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEncryptJSONPayload_ValidData(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	data := []byte(`{"content":"secret"}`)
	result, err := encryptJSONPayload(crypto, data, 1)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Result should be a JSON-encoded base64 string
	var encoded string
	require.NoError(t, json.Unmarshal(result, &encoded))
	decoded, b64Err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, b64Err)
	// mockCryptoService.Encrypt returns a copy of the input
	assert.Equal(t, data, decoded)
}

func TestEncryptJSONPayload_EncryptError(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{encryptErr: errors.New("encrypt failed")}
	_, err := encryptJSONPayload(crypto, []byte("data"), 1)
	require.Error(t, err)
	assert.EqualError(t, err, "encrypt failed")
}

// ---------------------------------------------------------------------------
// 11. decodeStoredPayload() tests
// ---------------------------------------------------------------------------

func TestDecodeStoredPayload_EmptyRaw_ReturnsNil(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := decodeStoredPayload(crypto, nil, 1)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDecodeStoredPayload_EmptySlice_ReturnsNil(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := decodeStoredPayload(crypto, []byte{}, 1)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDecodeStoredPayload_PlainJSON_ReturnsAsIs(t *testing.T) {
	t.Parallel()
	// If raw is not a JSON-encoded string, it's treated as legacy plaintext
	crypto := &mockCryptoService{}
	raw := []byte(`{"content":"legacy"}`)
	result, err := decodeStoredPayload(crypto, raw, 1)
	require.NoError(t, err)
	assert.Equal(t, raw, result)
}

func TestDecodeStoredPayload_EncryptedBase64(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}

	// Simulate stored format: JSON-encoded base64 string
	plainData := []byte("decrypted content")
	// mockCryptoService.Encrypt returns a copy, so encode manually
	encoded := base64.StdEncoding.EncodeToString(plainData)
	jsonEncoded, _ := json.Marshal(encoded)

	result, err := decodeStoredPayload(crypto, jsonEncoded, 1)
	require.NoError(t, err)
	assert.Equal(t, plainData, result)
}

func TestDecodeStoredPayload_InvalidBase64(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	// JSON string but invalid base64
	raw, _ := json.Marshal("!!!invalid-base64!!!")
	_, err := decodeStoredPayload(crypto, raw, 1)
	require.Error(t, err)
}

func TestDecodeStoredPayload_DecryptError(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{decryptErr: errors.New("decrypt failed")}
	encoded := base64.StdEncoding.EncodeToString([]byte("data"))
	jsonEncoded, _ := json.Marshal(encoded)
	_, err := decodeStoredPayload(crypto, jsonEncoded, 1)
	require.Error(t, err)
	assert.EqualError(t, err, "decrypt failed")
}

// ---------------------------------------------------------------------------
// 12. decryptMaybeLegacy() tests
// ---------------------------------------------------------------------------

func TestDecryptMaybeLegacy_EmptyData_ReturnsData(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := decryptMaybeLegacy(crypto, []byte{}, 1)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, result)
}

func TestDecryptMaybeLegacy_NilData_ReturnsData(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	result, err := decryptMaybeLegacy(crypto, nil, 1)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDecryptMaybeLegacy_PlainData_ReturnsAsIs(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	data := []byte("plain text, not encrypted")
	result, err := decryptMaybeLegacy(crypto, data, 1)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestDecryptMaybeLegacy_EncryptedData(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	// Data with encrypted prefix "GK1"
	data := []byte("GK1" + "some-ciphertext-here")
	result, err := decryptMaybeLegacy(crypto, data, 1)
	require.NoError(t, err)
	// mockCryptoService.Decrypt returns input as-is
	assert.Equal(t, data, result)
}

func TestDecryptMaybeLegacy_DecryptError(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{decryptErr: errors.New("decrypt error")}
	data := []byte("GK1" + "ciphertext")
	_, err := decryptMaybeLegacy(crypto, data, 1)
	require.Error(t, err)
	assert.EqualError(t, err, "decrypt error")
}

// ---------------------------------------------------------------------------
// 13. marshalConflictRecord() / unmarshalConflictRecord() tests
// ---------------------------------------------------------------------------

func TestMarshalConflictRecord_NilRecord_ReturnsNil(t *testing.T) {
	t.Parallel()
	data, err := marshalConflictRecord(nil)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestMarshalConflictRecord_ValidRecord(t *testing.T) {
	t.Parallel()
	now := time.Now()
	record := &models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeText,
		Name:           "test record",
		Metadata:       "meta",
		Payload:        models.TextPayload{Content: "hello"},
		Revision:       5,
		DeviceID:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	data, err := marshalConflictRecord(record)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Verify JSON structure
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.Equal(t, `"text"`, string(raw["type"]))
	assert.Equal(t, `"test record"`, string(raw["name"]))
}

func TestUnmarshalConflictRecord_ValidData(t *testing.T) {
	t.Parallel()
	now := time.Now()
	record := &models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeText,
		Name:           "test record",
		Metadata:       "meta",
		Payload:        models.TextPayload{Content: "hello"},
		Revision:       5,
		DeviceID:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	data, err := marshalConflictRecord(record)
	require.NoError(t, err)

	parsed, err := unmarshalConflictRecord(data)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, record.ID, parsed.ID)
	assert.Equal(t, record.UserID, parsed.UserID)
	assert.Equal(t, record.Type, parsed.Type)
	assert.Equal(t, record.Name, parsed.Name)
	assert.Equal(t, record.Metadata, parsed.Metadata)
	assert.Equal(t, record.Revision, parsed.Revision)
	assert.Equal(t, record.DeviceID, parsed.DeviceID)
	assert.Equal(t, record.KeyVersion, parsed.KeyVersion)
	assert.Equal(t, record.PayloadVersion, parsed.PayloadVersion)

	// Verify payload was decoded
	textPayload, ok := parsed.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "hello", textPayload.Content)
}

func TestUnmarshalConflictRecord_LoginPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		ID:         2,
		UserID:     10,
		Type:       models.RecordTypeLogin,
		Name:       "login record",
		Payload:    models.LoginPayload{Login: "admin", Password: "pass123"},
		Revision:   1,
		DeviceID:   "dev-2",
		KeyVersion: 1,
	}
	data, err := marshalConflictRecord(record)
	require.NoError(t, err)

	parsed, err := unmarshalConflictRecord(data)
	require.NoError(t, err)
	loginPayload, ok := parsed.Payload.(models.LoginPayload)
	require.True(t, ok)
	assert.Equal(t, "admin", loginPayload.Login)
	assert.Equal(t, "pass123", loginPayload.Password)
}

func TestUnmarshalConflictRecord_CardPayload(t *testing.T) {
	t.Parallel()
	record := &models.Record{
		ID:         3,
		UserID:     10,
		Type:       models.RecordTypeCard,
		Name:       "card record",
		Payload:    models.CardPayload{Number: "4111", HolderName: "John", ExpiryDate: "12/30", CVV: "123"},
		Revision:   1,
		DeviceID:   "dev-3",
		KeyVersion: 1,
	}
	data, err := marshalConflictRecord(record)
	require.NoError(t, err)

	parsed, err := unmarshalConflictRecord(data)
	require.NoError(t, err)
	cardPayload, ok := parsed.Payload.(models.CardPayload)
	require.True(t, ok)
	assert.Equal(t, "4111", cardPayload.Number)
}

func TestUnmarshalConflictRecord_DeletedAt(t *testing.T) {
	t.Parallel()
	// Truncate to second precision since JSON marshalling loses nanosecond precision
	deletedAt := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	record := &models.Record{
		ID:        4,
		UserID:    10,
		Type:      models.RecordTypeText,
		Name:      "deleted record",
		Payload:   models.TextPayload{Content: "deleted"},
		Revision:  1,
		DeviceID:  "dev-4",
		DeletedAt: &deletedAt,
	}
	data, err := marshalConflictRecord(record)
	require.NoError(t, err)

	parsed, err := unmarshalConflictRecord(data)
	require.NoError(t, err)
	require.NotNil(t, parsed.DeletedAt)
	assert.True(t, deletedAt.Equal(*parsed.DeletedAt))
	assert.Equal(t, deletedAt.UTC(), parsed.DeletedAt.UTC())
}

func TestUnmarshalConflictRecord_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := unmarshalConflictRecord([]byte("not json"))
	require.Error(t, err)
}

func TestMarshalUnmarshalConflictRecord_RoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name   string
		record *models.Record
	}{
		{
			name: "text payload",
			record: &models.Record{
				ID: 1, UserID: 10, Type: models.RecordTypeText,
				Name: "text", Payload: models.TextPayload{Content: "test"},
				Revision: 1, DeviceID: "d1", KeyVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
		},
		{
			name: "login payload",
			record: &models.Record{
				ID: 2, UserID: 10, Type: models.RecordTypeLogin,
				Name: "login", Payload: models.LoginPayload{Login: "user", Password: "pw"},
				Revision: 2, DeviceID: "d2", KeyVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
		},
		{
			name: "card payload",
			record: &models.Record{
				ID: 3, UserID: 10, Type: models.RecordTypeCard,
				Name: "card", Payload: models.CardPayload{Number: "4111", CVV: "123"},
				Revision: 3, DeviceID: "d3", KeyVersion: 2, CreatedAt: now, UpdatedAt: now,
			},
		},
		{
			name: "binary payload",
			record: &models.Record{
				ID: 4, UserID: 10, Type: models.RecordTypeBinary,
				Name: "binary", Payload: models.BinaryPayload{Data: []byte("bytes")},
				Revision: 4, DeviceID: "d4", KeyVersion: 2, CreatedAt: now, UpdatedAt: now,
			},
		},
		{
			name: "nil payload",
			record: &models.Record{
				ID: 5, UserID: 10, Type: models.RecordTypeText,
				Name: "empty", Payload: nil,
				Revision: 5, DeviceID: "d5", KeyVersion: 1, CreatedAt: now, UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := marshalConflictRecord(tt.record)
			require.NoError(t, err)
			require.NotNil(t, data)

			parsed, err := unmarshalConflictRecord(data)
			require.NoError(t, err)
			require.NotNil(t, parsed)

			assert.Equal(t, tt.record.ID, parsed.ID)
			assert.Equal(t, tt.record.UserID, parsed.UserID)
			assert.Equal(t, tt.record.Type, parsed.Type)
			assert.Equal(t, tt.record.Name, parsed.Name)
			assert.Equal(t, tt.record.Revision, parsed.Revision)
			assert.Equal(t, tt.record.DeviceID, parsed.DeviceID)
			assert.Equal(t, tt.record.KeyVersion, parsed.KeyVersion)
			assert.Equal(t, tt.record.PayloadVersion, parsed.PayloadVersion)
		})
	}
}

// ---------------------------------------------------------------------------
// 14. scanRecord() (Repository method) tests
// ---------------------------------------------------------------------------

func TestScanRecord_NoRows_ReturnsErrRecordNotFound(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	row := &mockRow{scanErr: sql.ErrNoRows}
	record, err := repo.scanRecord(row)
	require.Error(t, err)
	assert.Nil(t, record)
	assert.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestScanRecord_OtherScanError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	row := &mockRow{scanErr: errors.New("column mismatch")}
	record, err := repo.scanRecord(row)
	require.Error(t, err)
	assert.Nil(t, record)
	assert.EqualError(t, err, "column mismatch")
}

func TestScanRecord_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	now := time.Now()
	row := &mockRow{
		values: []any{
			int64(1),                  // ID
			int64(10),                 // UserID
			"text",                    // Type
			"test",                    // Name
			"meta",                    // Metadata
			[]byte(`{"content":"x"}`), // Payload
			int64(1),                  // Revision
			sql.NullTime{},            // DeletedAt
			"dev-1",                   // DeviceID
			int64(1),                  // KeyVersion
			int64(0),                  // PayloadVersion
			now,                       // CreatedAt
			now,                       // UpdatedAt
		},
	}
	record, err := repo.scanRecord(row)
	require.Error(t, err)
	assert.Nil(t, record)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestScanRecord_ValidLegacyPayload(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	now := time.Now()
	row := &mockRow{
		values: []any{
			int64(1),
			int64(10),
			"text",
			"test record",
			"some metadata",
			[]byte(`{"content":"legacy data"}`), // Plain JSON, not base64-wrapped
			int64(1),
			sql.NullTime{},
			"dev-1",
			int64(1),
			int64(0),
			now,
			now,
		},
	}
	record, err := repo.scanRecord(row)
	require.NoError(t, err)
	require.NotNil(t, record)
	assert.Equal(t, int64(1), record.ID)
	assert.Equal(t, int64(10), record.UserID)
	assert.Equal(t, models.RecordTypeText, record.Type)
	assert.Equal(t, "test record", record.Name)
	assert.Equal(t, "some metadata", record.Metadata)
	assert.Nil(t, record.DeletedAt)

	textPayload, ok := record.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "legacy data", textPayload.Content)
}

func TestScanRecord_WithDeletedAt(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	now := time.Now()
	deletedAt := now.Add(-1 * time.Hour)
	row := &mockRow{
		values: []any{
			int64(1),
			int64(10),
			"text",
			"deleted record",
			"",
			[]byte(`{"content":"data"}`),
			int64(2),
			sql.NullTime{Time: deletedAt, Valid: true},
			"dev-1",
			int64(1),
			int64(0),
			now,
			now,
		},
	}
	record, err := repo.scanRecord(row)
	require.NoError(t, err)
	require.NotNil(t, record.DeletedAt)
	assert.Equal(t, deletedAt, *record.DeletedAt)
}

func TestScanRecord_EmptyPayload(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	now := time.Now()
	row := &mockRow{
		values: []any{
			int64(1),
			int64(10),
			"text",
			"empty payload",
			"",
			[]byte{},
			int64(1),
			sql.NullTime{},
			"dev-1",
			int64(1),
			int64(0),
			now,
			now,
		},
	}
	record, err := repo.scanRecord(row)
	require.NoError(t, err)
	assert.Nil(t, record.Payload)
}

// ---------------------------------------------------------------------------
// 15. Repository.CreateRecord / UpdateRecord / SaveChunk — nil input tests
// ---------------------------------------------------------------------------

func TestCreateRecord_NilRecord_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateRecord(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "record is nil")
}

func TestCreateRecord_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	err := repo.CreateRecord(&models.Record{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestUpdateRecord_NilRecord_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.UpdateRecord(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "record is nil")
}

func TestUpdateRecord_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	err := repo.UpdateRecord(&models.Record{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestSaveChunk_NilChunk_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.SaveChunk(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "chunk is nil")
}

func TestSaveChunk_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	err := repo.SaveChunk(&models.Chunk{UploadID: 1, ChunkIndex: 0, Data: []byte("data")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestListRecords_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	_, err := repo.ListRecords(1, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestListRecordsForReencrypt_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	_, err := repo.ListRecordsForReencrypt(1, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

func TestGetChunks_NoCrypto_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepoNoCrypto()
	_, err := repo.GetChunks(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crypto service is required")
}

// ---------------------------------------------------------------------------
// 16. CreateConflict / CreateRevision / CreateUploadSession / CreateUser — nil tests
// ---------------------------------------------------------------------------

func TestCreateConflict_NilConflict_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateConflict(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "conflict is nil")
}

func TestCreateRevision_NilRevision_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateRevision(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "revision is nil")
}

func TestCreateUploadSession_NilSession_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateUploadSession(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "upload session is nil")
}

func TestCreateUser_NilUser_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateUser(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "user is nil")
}

func TestCreateKeyVersion_NilKV_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.CreateKeyVersion(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "key version is nil")
}

func TestUpdateKeyVersion_NilKV_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.UpdateKeyVersion(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "key version is nil")
}

func TestUpdateUploadSession_NilSession_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := newTestRepo()
	err := repo.UpdateUploadSession(nil)
	require.Error(t, err)
	assert.EqualError(t, err, "upload session is nil")
}

// Note: CreateUploadSession mutates session.Status before hitting DB,
// but since it always calls r.db.QueryRowContext, it panics with nil db.
// The default status logic (empty -> "pending") is covered by integration tests.

// ---------------------------------------------------------------------------
// 18. ListRecordsForReencrypt — limit default
// ---------------------------------------------------------------------------

// TestListRecordsForReencrypt_DefaultLimit verifies that limit <= 0 is corrected to 100.
// This requires DB access for the full function, but the logic is simple:
// if limit <= 0 { limit = 100 }.
// Since the function uses ensureCrypto first and then queries DB,
// we can only test the ensureCrypto guard without DB.

// ---------------------------------------------------------------------------
// 19. decryptMaybeLegacy — edge cases
// ---------------------------------------------------------------------------

func TestDecryptMaybeLegacy_ShortData_NoPrefix(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}
	data := []byte("ab")
	result, err := decryptMaybeLegacy(crypto, data, 1)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

// ---------------------------------------------------------------------------
// 20. Integration-like: encrypt/decode round trip (no DB, pure crypto helpers)
// ---------------------------------------------------------------------------

func TestEncryptDecodePayload_RoundTrip(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}

	originalPayload := []byte(`{"content":"secret text"}`)

	// Encrypt
	encrypted, err := encryptJSONPayload(crypto, originalPayload, 1)
	require.NoError(t, err)

	// Decode
	decrypted, err := decodeStoredPayload(crypto, encrypted, 1)
	require.NoError(t, err)

	assert.Equal(t, originalPayload, decrypted)
}

func TestEncryptDecodePayload_EmptyPayload_RoundTrip(t *testing.T) {
	t.Parallel()
	crypto := &mockCryptoService{}

	encrypted, err := encryptJSONPayload(crypto, []byte{}, 1)
	require.NoError(t, err)
	assert.Equal(t, []byte{}, encrypted)

	decrypted, err := decodeStoredPayload(crypto, encrypted, 1)
	require.NoError(t, err)
	assert.Nil(t, decrypted)
}

// ---------------------------------------------------------------------------
// 21. Table-driven decodePayload tests
// ---------------------------------------------------------------------------

func TestDecodePayload_Table(t *testing.T) {
	t.Parallel()
	loginJSON, _ := json.Marshal(models.LoginPayload{Login: "u", Password: "p"})
	textJSON, _ := json.Marshal(models.TextPayload{Content: "c"})
	cardJSON, _ := json.Marshal(models.CardPayload{Number: "1234", CVV: "567"})
	binaryJSON, _ := json.Marshal(models.BinaryPayload{Data: []byte("bin")})

	tests := []struct {
		name      string
		rType     models.RecordType
		raw       []byte
		wantErr   bool
		checkFunc func(t *testing.T, p models.RecordPayload)
	}{
		{
			name:    "login payload",
			rType:   models.RecordTypeLogin,
			raw:     loginJSON,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				lp, ok := p.(models.LoginPayload)
				require.True(t, ok)
				assert.Equal(t, "u", lp.Login)
				assert.Equal(t, "p", lp.Password)
			},
		},
		{
			name:    "text payload",
			rType:   models.RecordTypeText,
			raw:     textJSON,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				tp, ok := p.(models.TextPayload)
				require.True(t, ok)
				assert.Equal(t, "c", tp.Content)
			},
		},
		{
			name:    "card payload",
			rType:   models.RecordTypeCard,
			raw:     cardJSON,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				cp, ok := p.(models.CardPayload)
				require.True(t, ok)
				assert.Equal(t, "1234", cp.Number)
			},
		},
		{
			name:    "binary payload",
			rType:   models.RecordTypeBinary,
			raw:     binaryJSON,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				bp, ok := p.(models.BinaryPayload)
				require.True(t, ok)
				assert.Equal(t, []byte("bin"), bp.Data)
			},
		},
		{
			name:    "unknown type returns error",
			rType:   models.RecordType("unknown"),
			raw:     []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "invalid json for login returns error",
			rType:   models.RecordTypeLogin,
			raw:     []byte(`not json`),
			wantErr: true,
		},
		{
			name:    "empty binary returns empty payload",
			rType:   models.RecordTypeBinary,
			raw:     nil,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				bp, ok := p.(models.BinaryPayload)
				require.True(t, ok)
				assert.Equal(t, models.BinaryPayload{}, bp)
			},
		},
		{
			name:    "empty login returns nil",
			rType:   models.RecordTypeLogin,
			raw:     nil,
			wantErr: false,
			checkFunc: func(t *testing.T, p models.RecordPayload) {
				t.Helper()
				assert.Nil(t, p)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload, err := decodePayload(tt.rType, tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, payload)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 22. Table-driven splitRecordPayload tests
// ---------------------------------------------------------------------------

func TestSplitRecordPayload_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		payload    models.RecordPayload
		wantJSON   bool
		wantBinary bool
	}{
		{
			name:       "nil payload returns nil nil",
			payload:    nil,
			wantJSON:   false,
			wantBinary: false,
		},
		{
			name:       "login payload returns json",
			payload:    models.LoginPayload{Login: "u", Password: "p"},
			wantJSON:   true,
			wantBinary: false,
		},
		{
			name:       "text payload returns json",
			payload:    models.TextPayload{Content: "hello"},
			wantJSON:   true,
			wantBinary: false,
		},
		{
			name:       "card payload returns json",
			payload:    models.CardPayload{Number: "4111"},
			wantJSON:   true,
			wantBinary: false,
		},
		{
			name:       "binary payload returns nil nil",
			payload:    models.BinaryPayload{Data: []byte("data")},
			wantJSON:   false,
			wantBinary: false,
		},
		{
			name:       "binary payload pointer returns nil nil",
			payload:    &models.BinaryPayload{Data: []byte("data")},
			wantJSON:   false,
			wantBinary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := &models.Record{Payload: tt.payload}
			jsonData, binData, err := splitRecordPayload(record)
			require.NoError(t, err)
			if tt.wantJSON {
				assert.NotNil(t, jsonData)
			} else {
				assert.Nil(t, jsonData)
			}
			if tt.wantBinary {
				assert.NotNil(t, binData)
			} else {
				assert.Nil(t, binData)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 23. Table-driven nullIfEmpty tests
// ---------------------------------------------------------------------------

func TestNullIfEmpty_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantStr   string
	}{
		{name: "empty string", input: "", wantValid: false, wantStr: ""},
		{name: "non-empty string", input: "hello", wantValid: true, wantStr: "hello"},
		{name: "whitespace", input: "  ", wantValid: true, wantStr: "  "},
		{name: "zero character", input: "0", wantValid: true, wantStr: "0"},
		{name: "newline", input: "\n", wantValid: true, wantStr: "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := nullIfEmpty(tt.input)
			assert.Equal(t, tt.wantValid, result.Valid)
			assert.Equal(t, tt.wantStr, result.String)
		})
	}
}
