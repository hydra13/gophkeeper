package uploads

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/migrations"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories/database"
	"github.com/hydra13/gophkeeper/internal/repositories/file"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

// ---------------------------------------------------------------------------
// Integration test helpers
// ---------------------------------------------------------------------------

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GK_TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_DSN")
	}
	if dsn == "" {
		t.Skip("GK_TEST_DATABASE_DSN is not set")
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, migrations.Apply(db))
	require.NoError(t, truncateIntegrationTables(db))
	return db
}

func truncateIntegrationTables(db *sql.DB) error {
	_, err := db.Exec(`
		TRUNCATE TABLE
			upload_chunks,
			upload_sessions,
			payloads,
			sessions,
			sync_conflicts,
			record_revisions,
			records,
			key_versions,
			users
			RESTART IDENTITY CASCADE
	`)
	return err
}

type integrationEnv struct {
	svc        *Service
	repo       *database.Repository
	keyManager *keys.Manager
}

func setupIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()
	db := setupIntegrationDB(t)

	blobRepo, err := file.New(t.TempDir())
	require.NoError(t, err)

	repo, err := database.New(db, blobRepo)
	require.NoError(t, err)

	masterKey := make([]byte, 32)
	_, err = rand.Read(masterKey)
	require.NoError(t, err)

	keyManager, err := keys.NewManager(repo, base64.StdEncoding.EncodeToString(masterKey))
	require.NoError(t, err)

	cryptoService := cryptosvc.New(keyManager)
	repo.SetCrypto(cryptoService)

	svc, err := NewService(repo)
	require.NoError(t, err)

	return &integrationEnv{svc: svc, repo: repo, keyManager: keyManager}
}

func createIntegrationUser(t *testing.T, env *integrationEnv) *models.User {
	t.Helper()
	user := &models.User{
		Email:        fmt.Sprintf("user-%d@example.com", os.Getpid()),
		PasswordHash: "hash",
	}
	require.NoError(t, env.repo.CreateUser(user))
	return user
}

func createIntegrationKeyVersion(t *testing.T, env *integrationEnv) *models.KeyVersion {
	t.Helper()
	kv, err := env.keyManager.CreateActive()
	require.NoError(t, err)
	return kv
}

func createIntegrationRecord(t *testing.T, env *integrationEnv, userID int64, keyVersion int64) *models.Record {
	t.Helper()
	record := &models.Record{
		UserID:         userID,
		Type:           models.RecordTypeBinary,
		Name:           "test-binary",
		Metadata:       "meta",
		Payload:        models.BinaryPayload{},
		Revision:       1,
		DeviceID:       "device-test",
		KeyVersion:     keyVersion,
		PayloadVersion: 1,
	}
	require.NoError(t, env.repo.CreateRecord(record))
	return record
}

// ---------------------------------------------------------------------------
// Integration tests: Upload happy path
// ---------------------------------------------------------------------------

func TestIntegration_UploadHappyPath(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 256, 768, kv.Version)
	require.NoError(t, err)
	require.Greater(t, uploadID, int64(0))

	chunkData := [][]byte{
		[]byte("chunk-0-data"),
		[]byte("chunk-1-data"),
		[]byte("chunk-2-data"),
	}
	for i, data := range chunkData {
		received, total, completed, missing, uploadErr := env.svc.UploadChunk(uploadID, int64(i), data)
		require.NoError(t, uploadErr)
		require.Equal(t, int64(i+1), received)
		require.Equal(t, int64(3), total)
		if i < 2 {
			require.False(t, completed)
		} else {
			require.True(t, completed)
			require.Empty(t, missing)
		}
	}

	status, err := env.svc.GetUploadStatus(uploadID)
	require.NoError(t, err)
	require.Equal(t, "completed", status.Status)
	require.Equal(t, int64(3), status.ReceivedChunks)
}

// ---------------------------------------------------------------------------
// Integration tests: Download after upload
// ---------------------------------------------------------------------------

func TestIntegration_DownloadAfterUpload(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 256, 768, kv.Version)
	require.NoError(t, err)

	chunkData := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("gamma"),
	}
	for i, data := range chunkData {
		_, _, _, _, uploadErr := env.svc.UploadChunk(uploadID, int64(i), data)
		require.NoError(t, uploadErr)
	}

	for i, expected := range chunkData {
		resp, downloadErr := env.svc.DownloadChunk(uploadID, int64(i))
		require.NoError(t, downloadErr)
		require.Equal(t, expected, resp.Data)
		require.Equal(t, int64(i), resp.ChunkIndex)
		require.Equal(t, int64(3), resp.TotalChunks)
	}
}

// ---------------------------------------------------------------------------
// Integration tests: Upload resume
// ---------------------------------------------------------------------------

func TestIntegration_ResumeUpload(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 4, 512, 2048, kv.Version)
	require.NoError(t, err)

	// Upload first 2 chunks
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 0, []byte("part-0"))
	require.NoError(t, err)
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 1, []byte("part-1"))
	require.NoError(t, err)

	// Check status — should have missing chunks [2, 3]
	status, err := env.svc.GetUploadStatus(uploadID)
	require.NoError(t, err)
	require.Equal(t, "pending", status.Status)
	require.Equal(t, int64(2), status.ReceivedChunks)
	require.Equal(t, []int64{2, 3}, status.MissingChunks)

	// Resume: upload remaining chunks
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 2, []byte("part-2"))
	require.NoError(t, err)
	received, total, completed, missing, err := env.svc.UploadChunk(uploadID, 3, []byte("part-3"))
	require.NoError(t, err)
	require.Equal(t, int64(4), received)
	require.Equal(t, int64(4), total)
	require.True(t, completed)
	require.Empty(t, missing)

	// Download and verify all chunks
	for i := int64(0); i < 4; i++ {
		resp, downloadErr := env.svc.DownloadChunk(uploadID, i)
		require.NoError(t, downloadErr)
		require.Equal(t, []byte(fmt.Sprintf("part-%d", i)), resp.Data)
	}
}

// ---------------------------------------------------------------------------
// Integration tests: Download session (gRPC-style)
// ---------------------------------------------------------------------------

func TestIntegration_DownloadSessionFlow(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 100, 300, kv.Version)
	require.NoError(t, err)
	for i := int64(0); i < 3; i++ {
		_, _, _, _, uploadErr := env.svc.UploadChunk(uploadID, i, []byte(fmt.Sprintf("data-%d", i)))
		require.NoError(t, uploadErr)
	}

	download, err := env.svc.CreateDownloadSession(user.ID, record.ID)
	require.NoError(t, err)
	require.Equal(t, models.DownloadStatusActive, download.Status)
	require.Equal(t, int64(3), download.TotalChunks)

	for i := int64(0); i < 3; i++ {
		chunk, downloadErr := env.svc.DownloadChunkByID(download.ID, i)
		require.NoError(t, downloadErr)
		require.Equal(t, []byte(fmt.Sprintf("data-%d", i)), chunk.Data)
	}

	for i := int64(0); i < 3; i++ {
		confirmed, total, dlStatus, confirmErr := env.svc.ConfirmChunk(download.ID, i)
		require.NoError(t, confirmErr)
		require.Equal(t, i+1, confirmed)
		require.Equal(t, int64(3), total)
		if i < 2 {
			require.Equal(t, models.DownloadStatusActive, dlStatus)
		} else {
			require.Equal(t, models.DownloadStatusCompleted, dlStatus)
		}
	}

	finalStatus, err := env.svc.GetDownloadStatus(download.ID)
	require.NoError(t, err)
	require.Equal(t, models.DownloadStatusCompleted, finalStatus.Status)
	require.Equal(t, int64(3), finalStatus.ConfirmedChunks)
}

// ---------------------------------------------------------------------------
// Integration tests: Download resume
// ---------------------------------------------------------------------------

func TestIntegration_DownloadResume(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 100, 300, kv.Version)
	require.NoError(t, err)
	for i := int64(0); i < 3; i++ {
		_, _, _, _, uploadErr := env.svc.UploadChunk(uploadID, i, []byte(fmt.Sprintf("resume-%d", i)))
		require.NoError(t, uploadErr)
	}

	download, err := env.svc.CreateDownloadSession(user.ID, record.ID)
	require.NoError(t, err)

	// Confirm only first chunk
	confirmed, _, _, err := env.svc.ConfirmChunk(download.ID, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), confirmed)

	// Simulate disconnect: check remaining chunks
	dlStatus, err := env.svc.GetDownloadStatus(download.ID)
	require.NoError(t, err)
	require.Equal(t, models.DownloadStatusActive, dlStatus.Status)
	require.Equal(t, []int64{1, 2}, dlStatus.RemainingChunks())

	// Resume: download and confirm remaining
	for i := int64(1); i < 3; i++ {
		chunk, downloadErr := env.svc.DownloadChunkByID(download.ID, i)
		require.NoError(t, downloadErr)
		require.Equal(t, []byte(fmt.Sprintf("resume-%d", i)), chunk.Data)

		confirmed, total, dlStatus2, confirmErr := env.svc.ConfirmChunk(download.ID, i)
		require.NoError(t, confirmErr)
		require.Equal(t, i+1, confirmed)
		require.Equal(t, int64(3), total)
		if i < 2 {
			require.Equal(t, models.DownloadStatusActive, dlStatus2)
		} else {
			require.Equal(t, models.DownloadStatusCompleted, dlStatus2)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration tests: Error scenarios
// ---------------------------------------------------------------------------

func TestIntegration_UploadChunkOutOfOrder(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 256, 768, kv.Version)
	require.NoError(t, err)

	// Try uploading chunk 2 before 0 and 1
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 2, []byte("skip"))
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestIntegration_UploadDuplicateChunk(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 256, 768, kv.Version)
	require.NoError(t, err)

	_, _, _, _, err = env.svc.UploadChunk(uploadID, 0, []byte("first"))
	require.NoError(t, err)

	// Try uploading chunk 0 again — out of order because expected is now 1
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 0, []byte("duplicate"))
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestIntegration_DownloadFromIncompleteUpload(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 256, 768, kv.Version)
	require.NoError(t, err)

	_, _, _, _, err = env.svc.UploadChunk(uploadID, 0, []byte("partial"))
	require.NoError(t, err)

	// Try downloading from non-completed upload
	_, err = env.svc.DownloadChunk(uploadID, 0)
	require.ErrorIs(t, err, models.ErrUploadNotPending)
}

func TestIntegration_UploadChunkOutOfRange(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 2, 256, 512, kv.Version)
	require.NoError(t, err)

	_, _, _, _, err = env.svc.UploadChunk(uploadID, 5, []byte("overflow"))
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestIntegration_UploadToCompletedSession(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 1, 256, 256, kv.Version)
	require.NoError(t, err)

	_, _, _, _, err = env.svc.UploadChunk(uploadID, 0, []byte("only-one"))
	require.NoError(t, err)

	// Try uploading after completion
	_, _, _, _, err = env.svc.UploadChunk(uploadID, 1, []byte("extra"))
	require.ErrorIs(t, err, models.ErrUploadCompleted)
}

func TestIntegration_DownloadSessionChunkOutOfRange(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 2, 256, 512, kv.Version)
	require.NoError(t, err)
	for i := int64(0); i < 2; i++ {
		_, _, _, _, uploadErr := env.svc.UploadChunk(uploadID, i, []byte("x"))
		require.NoError(t, uploadErr)
	}

	download, err := env.svc.CreateDownloadSession(user.ID, record.ID)
	require.NoError(t, err)

	_, err = env.svc.DownloadChunkByID(download.ID, 10)
	require.ErrorIs(t, err, models.ErrChunkOutOfRange)
}

func TestIntegration_ConfirmChunkOutOfOrder(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	uploadID, err := env.svc.CreateSession(user.ID, record.ID, 3, 100, 300, kv.Version)
	require.NoError(t, err)
	for i := int64(0); i < 3; i++ {
		_, _, _, _, uploadErr := env.svc.UploadChunk(uploadID, i, []byte("x"))
		require.NoError(t, uploadErr)
	}

	download, err := env.svc.CreateDownloadSession(user.ID, record.ID)
	require.NoError(t, err)

	// Try confirming chunk 2 before 0
	_, _, _, err = env.svc.ConfirmChunk(download.ID, 2)
	require.ErrorIs(t, err, models.ErrChunkOutOfOrder)
}

func TestIntegration_CreateDownloadSessionNoCompletedUpload(t *testing.T) {
	env := setupIntegrationEnv(t)

	user := createIntegrationUser(t, env)
	kv := createIntegrationKeyVersion(t, env)
	record := createIntegrationRecord(t, env, user.ID, kv.Version)

	// Don't create any upload session — should fail
	_, err := env.svc.CreateDownloadSession(user.ID, record.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "find completed upload")
}
