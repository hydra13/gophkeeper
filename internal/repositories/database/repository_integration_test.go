package database

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/migrations"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/repositories/file"
)

func TestMigrationsApply(t *testing.T) {
	db := setupDB(t)
	row := db.QueryRow(`SELECT to_regclass('public.users')`)
	var name sql.NullString
	require.NoError(t, row.Scan(&name))
	require.True(t, name.Valid)
	require.Equal(t, "users", name.String)
}

func TestUserUniqueEmail(t *testing.T) {
	db := setupDB(t)
	repo := newRepository(t, db)

	user := &models.User{
		Email:        "user@example.com",
		PasswordHash: "hash",
	}
	require.NoError(t, repo.CreateUser(user))

	duplicate := &models.User{
		Email:        "user@example.com",
		PasswordHash: "hash2",
	}
	require.ErrorIs(t, repo.CreateUser(duplicate), models.ErrEmailAlreadyExists)
}

func TestRecordSoftDelete(t *testing.T) {
	db := setupDB(t)
	repo := newRepository(t, db)

	user := createUser(t, repo)
	keyVersion := createKeyVersion(t, repo, 1, models.KeyStatusActive)

	record := &models.Record{
		UserID:     user.ID,
		Type:       models.RecordTypeText,
		Name:       "secret",
		Metadata:   "meta",
		Payload:    models.TextPayload{Content: "data"},
		Revision:   1,
		DeviceID:   "device-1",
		KeyVersion: keyVersion.Version,
	}
	require.NoError(t, repo.CreateRecord(record))

	require.NoError(t, repo.DeleteRecord(record.ID))

	stored, err := repo.GetRecord(record.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.DeletedAt)

	records, err := repo.ListRecords(user.ID)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestUploadResumeState(t *testing.T) {
	db := setupDB(t)
	repo := newRepository(t, db)

	user := createUser(t, repo)
	keyVersion := createKeyVersion(t, repo, 1, models.KeyStatusActive)

	record := &models.Record{
		UserID:         user.ID,
		Type:           models.RecordTypeBinary,
		Name:           "binary",
		Metadata:       "meta",
		Payload:        models.BinaryPayload{},
		Revision:       1,
		DeviceID:       "device-1",
		KeyVersion:     keyVersion.Version,
		PayloadVersion: 1,
	}
	require.NoError(t, repo.CreateRecord(record))

	upload := &models.UploadSession{
		RecordID:     record.ID,
		UserID:       user.ID,
		Status:       models.UploadStatusPending,
		TotalChunks:  3,
		ChunkSize:    5,
		TotalSize:    15,
		ReceivedChunks: 0,
		KeyVersion:   keyVersion.Version,
	}
	require.NoError(t, repo.CreateUploadSession(upload))

	require.NoError(t, repo.SaveChunk(&models.Chunk{
		UploadID:   upload.ID,
		ChunkIndex: 0,
		Data:       []byte("chunk"),
	}))
	require.NoError(t, repo.SaveChunk(&models.Chunk{
		UploadID:   upload.ID,
		ChunkIndex: 2,
		Data:       []byte("chunk"),
	}))

	session, err := repo.GetUploadSession(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), session.ReceivedChunks)
	require.ElementsMatch(t, []int64{1}, session.MissingChunks())
}

func TestRevisionUniqueConstraint(t *testing.T) {
	db := setupDB(t)
	repo := newRepository(t, db)

	user := createUser(t, repo)
	keyVersion := createKeyVersion(t, repo, 1, models.KeyStatusActive)

	record := &models.Record{
		UserID:     user.ID,
		Type:       models.RecordTypeText,
		Name:       "secret",
		Metadata:   "meta",
		Payload:    models.TextPayload{Content: "data"},
		Revision:   1,
		DeviceID:   "device-1",
		KeyVersion: keyVersion.Version,
	}
	require.NoError(t, repo.CreateRecord(record))

	rev := &models.RecordRevision{
		RecordID: record.ID,
		UserID:   user.ID,
		Revision: 1,
		DeviceID: "device-1",
	}
	require.NoError(t, repo.CreateRevision(rev))

	duplicate := &models.RecordRevision{
		RecordID: record.ID,
		UserID:   user.ID,
		Revision: 1,
		DeviceID: "device-2",
	}
	require.ErrorIs(t, repo.CreateRevision(duplicate), models.ErrRevisionConflict)
}

func setupDB(t *testing.T) *sql.DB {
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
	t.Cleanup(func() {
		_ = db.Close()
	})

	require.NoError(t, migrations.Apply(db))
	require.NoError(t, truncateTables(db))
	return db
}

func truncateTables(db *sql.DB) error {
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

func newRepository(t *testing.T, db *sql.DB) *Repository {
	t.Helper()
	blobRepo, err := file.New(t.TempDir())
	require.NoError(t, err)
	repo, err := New(db, blobRepo)
	require.NoError(t, err)
	return repo
}

func createUser(t *testing.T, repo *Repository) *models.User {
	t.Helper()
	user := &models.User{
		Email:        "user-" + time.Now().Format("150405.000") + "@example.com",
		PasswordHash: "hash",
	}
	require.NoError(t, repo.CreateUser(user))
	return user
}

func createKeyVersion(t *testing.T, repo *Repository, version int64, status models.KeyStatus) *models.KeyVersion {
	t.Helper()
	kv := &models.KeyVersion{
		Version: version,
		Status:  status,
	}
	require.NoError(t, repo.CreateKeyVersion(kv))
	return kv
}
