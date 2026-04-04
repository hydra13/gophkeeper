//go:build integration
// +build integration

package reencrypt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/migrations"
	"github.com/hydra13/gophkeeper/internal/models"
	dbrepo "github.com/hydra13/gophkeeper/internal/repositories/database"
	"github.com/hydra13/gophkeeper/internal/repositories/file"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

func TestJobReencrypt(t *testing.T) {
	db := setupDB(t)
	blobRepo, err := file.New(t.TempDir())
	require.NoError(t, err)
	repo, err := dbrepo.New(db, blobRepo)
	require.NoError(t, err)

	masterKey := testMasterKey(t)
	keyManager, err := keys.NewManager(repo, masterKey)
	require.NoError(t, err)
	cryptoService := cryptosvc.New(keyManager)
	repo.SetCrypto(cryptoService)

	active, err := keyManager.CreateActive()
	require.NoError(t, err)

	user := &models.User{Email: "user@example.com", PasswordHash: "hash"}
	require.NoError(t, repo.CreateUser(user))

	record := &models.Record{
		UserID:     user.ID,
		Type:       models.RecordTypeText,
		Name:       "secret",
		Metadata:   "meta",
		Payload:    models.TextPayload{Content: "data"},
		Revision:   1,
		DeviceID:   "device-1",
		KeyVersion: active.Version,
	}
	require.NoError(t, repo.CreateRecord(record))

	newActive, err := keyManager.Rotate()
	require.NoError(t, err)

	job := New(WithDeps(repo, blobRepo, cryptoService, keyManager), WithBatchSize(10))
	require.NoError(t, job.runOnce(context.Background()))

	stored, err := repo.GetRecord(record.ID)
	require.NoError(t, err)
	require.Equal(t, newActive.Version, stored.KeyVersion)
	payload := stored.Payload.(models.TextPayload)
	require.Equal(t, "data", payload.Content)
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

func testMasterKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(key)
}
