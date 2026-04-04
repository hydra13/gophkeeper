//go:build integration
// +build integration

package sync

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/migrations"
	"github.com/hydra13/gophkeeper/internal/models"
	dbrepo "github.com/hydra13/gophkeeper/internal/repositories/database"
	cryptosvc "github.com/hydra13/gophkeeper/internal/services/crypto"
	"github.com/hydra13/gophkeeper/internal/services/keys"
	"github.com/hydra13/gophkeeper/internal/storage"
)

func TestSyncPushPullHappyPath(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "my secret",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "hello"},
			},
			BaseRevision: 0,
		},
	}

	accepted, conflicts, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	require.Len(t, accepted, 1)
	require.Empty(t, conflicts)
	require.Equal(t, int64(1), accepted[0].Revision)

	revs, records, _, err := svc.Pull(user.ID, "device-1", 0, 50)
	require.NoError(t, err)
	require.Len(t, revs, 1)
	require.Len(t, records, 1)
	require.Equal(t, "my secret", records[0].Name)
}

func TestSyncConflictOnConcurrentUpdate(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись через sync push
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Первое обновление с правильной base_revision — успешно
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "updated-v1",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "v1"},
			},
			BaseRevision: baseRev,
		},
	}
	accepted2, conflicts2, err := svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)
	require.Len(t, accepted2, 1)
	require.Empty(t, conflicts2)

	// Второе обновление с устаревшей base_revision — конфликт
	conflictChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "updated-v2",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "v2"},
			},
			BaseRevision: baseRev, // устаревшая
		},
	}
	_, conflicts3, err := svc.Push(user.ID, "device-1", conflictChanges)
	require.NoError(t, err)
	require.Len(t, conflicts3, 1)
	require.Equal(t, recordID, conflicts3[0].RecordID)
	require.False(t, conflicts3[0].Resolved)
}

func TestSyncSoftDelete(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "to-delete",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "data"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID

	// Удаляем через sync
	deleteChanges := []models.PendingChange{
		{
			Record:       &models.Record{ID: recordID},
			Deleted:      true,
			BaseRevision: 0,
		},
	}
	acceptedDel, _, err := svc.Push(user.ID, "device-1", deleteChanges)
	require.NoError(t, err)
	require.Len(t, acceptedDel, 1)

	// Pull проверяет что запись удалена
	_, records, _, err := svc.Pull(user.ID, "device-2", 0, 50)
	require.NoError(t, err)
	found := false
	for _, r := range records {
		if r.ID == recordID {
			found = true
			require.NotNil(t, r.DeletedAt, "record should be soft deleted")
		}
	}
	require.True(t, found, "deleted record should be in pull results")
}

func TestSyncIncrementalPull(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем 3 записи последовательно
	for i := 0; i < 3; i++ {
		changes := []models.PendingChange{
			{
				Record: &models.Record{
					UserID:     user.ID,
					Type:       models.RecordTypeText,
					Name:       "record-" + time.Now().Format("150405.000"),
					DeviceID:   "device-1",
					KeyVersion: 1,
					Payload:    models.TextPayload{Content: "data"},
				},
				BaseRevision: 0,
			},
		}
		_, _, err := svc.Push(user.ID, "device-1", changes)
		require.NoError(t, err)
	}

	// Pull with limit=1 — should get first revision, hasMore=true
	revs1, _, _, err := svc.Pull(user.ID, "device-1", 0, 1)
	require.NoError(t, err)
	require.Len(t, revs1, 1)
	// hasMore logic is in HTTP handler; service returns all revisions up to limit

	// Pull since_revision=1
	revs2, _, _, err := svc.Pull(user.ID, "device-1", 1, 50)
	require.NoError(t, err)
	require.Len(t, revs2, 2) // revisions 2 and 3
}

func TestSyncResolveConflictLocal(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Обновляем на сервере
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "server-update",
				DeviceID:   "device-2",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "server"},
			},
			BaseRevision: baseRev,
		},
	}
	_, _, err = svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)

	// Провоцируем конфликт
	conflictChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "local-update",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "local"},
			},
			BaseRevision: baseRev,
		},
	}
	_, conflicts, err := svc.Push(user.ID, "device-1", conflictChanges)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Разрешаем конфликт в пользу локальной версии
	record, err := svc.ResolveConflict(user.ID, conflicts[0].ID, models.ConflictResolutionLocal)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "local-update", record.Name)

	// Конфликт должен быть разрешён
	conflictsAfter, err := svc.GetConflicts(user.ID)
	require.NoError(t, err)
	require.Empty(t, conflictsAfter)
}

func TestSyncResolveConflictServer(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Обновляем на сервере
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "server-update",
				DeviceID:   "device-2",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "server"},
			},
			BaseRevision: baseRev,
		},
	}
	_, _, err = svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)

	// Провоцируем конфликт
	conflictChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "local-update",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "local"},
			},
			BaseRevision: baseRev,
		},
	}
	_, conflicts, err := svc.Push(user.ID, "device-1", conflictChanges)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Разрешаем в пользу сервера
	record, err := svc.ResolveConflict(user.ID, conflicts[0].ID, models.ConflictResolutionServer)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "server-update", record.Name)
}

func TestSyncRestoreAfterSoftDelete(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "will-restore",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "data"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID

	// Удаляем через sync
	deleteChanges := []models.PendingChange{
		{
			Record:  &models.Record{ID: recordID},
			Deleted: true,
		},
	}
	_, _, err = svc.Push(user.ID, "device-1", deleteChanges)
	require.NoError(t, err)

	// Получаем текущую ревизию записи
	record, err := repo.GetRecord(recordID)
	require.NoError(t, err)
	require.NotNil(t, record.DeletedAt)
	currentRev := record.Revision

	// Восстанавливаем через push update (запись с cleared deleted_at)
	restoreChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "restored",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "restored"},
			},
			BaseRevision: currentRev,
		},
	}
	acceptedRestore, conflicts, err := svc.Push(user.ID, "device-1", restoreChanges)
	require.NoError(t, err)
	require.Len(t, acceptedRestore, 1)
	require.Empty(t, conflicts)

	// Проверяем что запись восстановлена
	restored, err := repo.GetRecord(recordID)
	require.NoError(t, err)
	require.Nil(t, restored.DeletedAt, "record should be restored (deleted_at should be nil)")
	require.Equal(t, "restored", restored.Name)
}

// Helpers

func setupSyncService(t *testing.T) (*Service, *dbrepo.Repository, *keys.Manager) {
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
	require.NoError(t, truncateAllTables(db))

	blobRepo, err := storage.NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	repo, err := dbrepo.New(db, blobRepo)
	require.NoError(t, err)

	masterKey := testMasterKey(t)
	keyManager, err := keys.NewManager(repo, masterKey)
	require.NoError(t, err)
	cryptoService := cryptosvc.New(keyManager)
	repo.SetCrypto(cryptoService)

	svc, err := NewService(repo, repo)
	require.NoError(t, err)

	return svc, repo, keyManager
}

func createUserAndKey(t *testing.T, repo *dbrepo.Repository, km *keys.Manager) (*models.User, *models.KeyVersion) {
	t.Helper()
	user := &models.User{
		Email:        "sync-user-" + time.Now().Format("150405.000") + "@example.com",
		PasswordHash: "hash",
	}
	require.NoError(t, repo.CreateUser(user))
	kv, err := km.CreateActive()
	require.NoError(t, err)
	return user, kv
}

func truncateAllTables(db *sql.DB) error {
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

func TestSyncStaleDeleteAfterUpdateReturnsConflict(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Обновляем на device-2
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "updated-v2",
				DeviceID:   "device-2",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "v2"},
			},
			BaseRevision: baseRev,
		},
	}
	accepted2, _, err := svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)
	require.Len(t, accepted2, 1)

	// Удаляем с устаревшей base_revision с device-1 — должен быть конфликт
	deleteChanges := []models.PendingChange{
		{
			Record:       &models.Record{ID: recordID},
			Deleted:      true,
			BaseRevision: baseRev, // устаревшая ревизия
		},
	}
	_, conflicts, err := svc.Push(user.ID, "device-1", deleteChanges)
	require.NoError(t, err)
	require.Len(t, conflicts, 1, "stale delete should produce a conflict")
	require.Equal(t, recordID, conflicts[0].RecordID)

	// Запись НЕ должна быть удалена
	record, err := repo.GetRecord(recordID)
	require.NoError(t, err)
	require.Nil(t, record.DeletedAt, "record should not be deleted on stale delete conflict")
	require.Equal(t, "updated-v2", record.Name)
}

func TestSyncDeleteThenRestoreViaUpdate(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "to-restore",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "data"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID

	// Soft delete через sync
	deleteChanges := []models.PendingChange{
		{
			Record:       &models.Record{ID: recordID},
			Deleted:      true,
			BaseRevision: 0,
		},
	}
	_, _, err = svc.Push(user.ID, "device-1", deleteChanges)
	require.NoError(t, err)

	// Получаем текущую ревизию удалённой записи
	record, err := repo.GetRecord(recordID)
	require.NoError(t, err)
	require.NotNil(t, record.DeletedAt)
	currentRev := record.Revision

	// Восстанавливаем через push update с актуальной BaseRevision
	restoreChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "restored-name",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "restored"},
			},
			BaseRevision: currentRev,
		},
	}
	acceptedRestore, conflicts, err := svc.Push(user.ID, "device-1", restoreChanges)
	require.NoError(t, err)
	require.Len(t, acceptedRestore, 1)
	require.Empty(t, conflicts)

	// Проверяем, что запись восстановлена
	restored, err := repo.GetRecord(recordID)
	require.NoError(t, err)
	require.Nil(t, restored.DeletedAt, "record should be restored (deleted_at should be nil)")
	require.Equal(t, "restored-name", restored.Name)
}

func TestSyncDeleteConflictReturnsBothVersions(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Обновляем на device-2
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "server-version",
				DeviceID:   "device-2",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "server"},
			},
			BaseRevision: baseRev,
		},
	}
	_, _, err = svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)

	// Пытаемся удалить с устаревшей ревизией
	deleteChanges := []models.PendingChange{
		{
			Record:       &models.Record{ID: recordID},
			Deleted:      true,
			BaseRevision: baseRev,
		},
	}
	_, conflicts, err := svc.Push(user.ID, "device-1", deleteChanges)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Проверяем, что в конфликте есть серверная версия записи
	require.NotNil(t, conflicts[0].ServerRecord, "conflict should have server record")
	require.Equal(t, "server-version", conflicts[0].ServerRecord.Name)

	// Проверяем, что конфликт доступен через GetConflicts со снимками
	allConflicts, err := svc.GetConflicts(user.ID)
	require.NoError(t, err)
	require.Len(t, allConflicts, 1)
	require.NotNil(t, allConflicts[0].ServerRecord, "GetConflicts should return server record snapshot")
	require.Equal(t, "server-version", allConflicts[0].ServerRecord.Name)
}

func TestSyncUpdateConflictReturnsBothVersions(t *testing.T) {
	svc, repo, km := setupSyncService(t)
	user, _ := createUserAndKey(t, repo, km)

	// Создаем запись
	changes := []models.PendingChange{
		{
			Record: &models.Record{
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "original",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "original"},
			},
			BaseRevision: 0,
		},
	}
	accepted, _, err := svc.Push(user.ID, "device-1", changes)
	require.NoError(t, err)
	recordID := accepted[0].RecordID
	baseRev := accepted[0].Revision

	// Обновляем на device-2
	updateChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "server-version",
				DeviceID:   "device-2",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "server"},
			},
			BaseRevision: baseRev,
		},
	}
	_, _, err = svc.Push(user.ID, "device-2", updateChanges)
	require.NoError(t, err)

	// Конфликт update с device-1
	conflictChanges := []models.PendingChange{
		{
			Record: &models.Record{
				ID:         recordID,
				UserID:     user.ID,
				Type:       models.RecordTypeText,
				Name:       "local-version",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Payload:    models.TextPayload{Content: "local"},
			},
			BaseRevision: baseRev,
		},
	}
	_, conflicts, err := svc.Push(user.ID, "device-1", conflictChanges)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	// Обе версии должны быть доступны
	require.NotNil(t, conflicts[0].LocalRecord, "conflict should have local record")
	require.NotNil(t, conflicts[0].ServerRecord, "conflict should have server record")
	require.Equal(t, "local-version", conflicts[0].LocalRecord.Name)
	require.Equal(t, "server-version", conflicts[0].ServerRecord.Name)

	// Проверяем через GetConflicts
	allConflicts, err := svc.GetConflicts(user.ID)
	require.NoError(t, err)
	require.Len(t, allConflicts, 1)
	require.NotNil(t, allConflicts[0].LocalRecord)
	require.NotNil(t, allConflicts[0].ServerRecord)
	require.Equal(t, "local-version", allConflicts[0].LocalRecord.Name)
	require.Equal(t, "server-version", allConflicts[0].ServerRecord.Name)
}
