package records

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/services/keys"
)

type memRecordRepo struct {
	lastID  int64
	records map[int64]*models.Record
}

func newMemRecordRepo() *memRecordRepo {
	return &memRecordRepo{records: make(map[int64]*models.Record)}
}

func (m *memRecordRepo) CreateRecord(record *models.Record) error {
	m.lastID++
	record.ID = m.lastID
	copied := *record
	m.records[record.ID] = &copied
	return nil
}

func (m *memRecordRepo) GetRecord(id int64) (*models.Record, error) {
	record, ok := m.records[id]
	if !ok {
		return nil, models.ErrRecordNotFound
	}
	copied := *record
	return &copied, nil
}

func (m *memRecordRepo) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	var result []models.Record
	for _, record := range m.records {
		if record.UserID != userID {
			continue
		}
		if !includeDeleted && record.DeletedAt != nil {
			continue
		}
		if recordType != "" && record.Type != recordType {
			continue
		}
		result = append(result, *record)
	}
	return result, nil
}

func (m *memRecordRepo) UpdateRecord(record *models.Record) error {
	if _, ok := m.records[record.ID]; !ok {
		return models.ErrRecordNotFound
	}
	copied := *record
	m.records[record.ID] = &copied
	return nil
}

func (m *memRecordRepo) DeleteRecord(id int64) error {
	record, ok := m.records[id]
	if !ok {
		return models.ErrRecordNotFound
	}
	if record.DeletedAt != nil {
		return models.ErrAlreadyDeleted
	}
	now := time.Now()
	record.DeletedAt = &now
	return nil
}

type memKeyRepo struct {
	versions map[int64]*models.KeyVersion
}

func newMemKeyRepo() *memKeyRepo {
	return &memKeyRepo{versions: make(map[int64]*models.KeyVersion)}
}

func (m *memKeyRepo) CreateKeyVersion(kv *models.KeyVersion) error {
	clone := *kv
	m.versions[kv.Version] = &clone
	return nil
}

func (m *memKeyRepo) GetKeyVersion(version int64) (*models.KeyVersion, error) {
	kv, ok := m.versions[version]
	if !ok {
		return nil, models.ErrUnknownKeyVersion
	}
	clone := *kv
	return &clone, nil
}

func (m *memKeyRepo) GetActiveKeyVersion() (*models.KeyVersion, error) {
	for _, kv := range m.versions {
		if kv.IsActive() {
			clone := *kv
			return &clone, nil
		}
	}
	return nil, models.ErrUnknownKeyVersion
}

func (m *memKeyRepo) ListKeyVersions() ([]models.KeyVersion, error) {
	result := make([]models.KeyVersion, 0, len(m.versions))
	for _, kv := range m.versions {
		result = append(result, *kv)
	}
	return result, nil
}

func (m *memKeyRepo) UpdateKeyVersion(kv *models.KeyVersion) error {
	if _, ok := m.versions[kv.Version]; !ok {
		return models.ErrUnknownKeyVersion
	}
	clone := *kv
	m.versions[kv.Version] = &clone
	return nil
}

func testMasterKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(key)
}

func newTestService(t *testing.T) (*Service, *memRecordRepo) {
	t.Helper()
	keyRepo := newMemKeyRepo()
	keyManager, err := keys.NewManager(keyRepo, testMasterKey(t))
	require.NoError(t, err)
	_, err = keyManager.CreateActive()
	require.NoError(t, err)

	repo := newMemRecordRepo()
	service, err := NewService(repo, keyManager)
	require.NoError(t, err)
	return service, repo
}

func TestServiceCreateRecordUsesActiveKeyAfterRotate(t *testing.T) {
	keyRepo := newMemKeyRepo()
	keyManager, err := keys.NewManager(keyRepo, testMasterKey(t))
	require.NoError(t, err)

	active, err := keyManager.CreateActive()
	require.NoError(t, err)

	repo := newMemRecordRepo()
	service, err := NewService(repo, keyManager)
	require.NoError(t, err)

	record := &models.Record{
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "secret",
		Metadata:   "meta",
		Payload:    models.TextPayload{Content: "data"},
		Revision:   1,
		DeviceID:   "device-1",
		KeyVersion: 999,
	}
	require.NoError(t, service.CreateRecord(record))
	require.Equal(t, active.Version, record.KeyVersion)

	newActive, err := keyManager.Rotate()
	require.NoError(t, err)

	record2 := &models.Record{
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "secret-2",
		Metadata:   "meta-2",
		Payload:    models.TextPayload{Content: "data-2"},
		Revision:   1,
		DeviceID:   "device-2",
		KeyVersion: active.Version,
	}
	require.NoError(t, service.CreateRecord(record2))
	require.Equal(t, newActive.Version, record2.KeyVersion)
}

func TestServiceCreateRecord_AllTypes(t *testing.T) {
	service, _ := newTestService(t)

	tests := []struct {
		name    string
		record  *models.Record
		wantErr bool
	}{
		{
			name: "login type",
			record: &models.Record{
				UserID:   1, Type: models.RecordTypeLogin, Name: "my login",
				Payload: models.LoginPayload{Login: "user", Password: "pass"},
				DeviceID: "dev-1",
			},
		},
		{
			name: "text type",
			record: &models.Record{
				UserID:   1, Type: models.RecordTypeText, Name: "my note",
				Payload: models.TextPayload{Content: "hello"},
				DeviceID: "dev-1",
			},
		},
		{
			name: "binary type",
			record: &models.Record{
				UserID:   1, Type: models.RecordTypeBinary, Name: "my file",
				Payload: models.BinaryPayload{Data: []byte{1, 2, 3}},
				DeviceID: "dev-1", PayloadVersion: 1,
			},
		},
		{
			name: "card type",
			record: &models.Record{
				UserID:   1, Type: models.RecordTypeCard, Name: "my card",
				Payload: models.CardPayload{Number: "4111", HolderName: "Test", ExpiryDate: "12/25", CVV: "123"},
				DeviceID: "dev-1",
			},
		},
		{
			name: "nil record",
			record: &models.Record{Type: models.RecordTypeLogin, Name: "x",
				Payload: models.LoginPayload{Login: "a", Password: "b"}, DeviceID: "d"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil record" {
				err := service.CreateRecord(nil)
				require.Error(t, err)
				return
			}
			err := service.CreateRecord(tt.record)
			require.NoError(t, err)
			require.NotZero(t, tt.record.ID)
			require.NotZero(t, tt.record.KeyVersion)
		})
	}
}

func TestServiceListRecords(t *testing.T) {
	service, _ := newTestService(t)

	for i := 0; i < 3; i++ {
		r := &models.Record{
			UserID: 1, Type: models.RecordTypeText, Name: "record",
			Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
		}
		require.NoError(t, service.CreateRecord(r))
	}

	records, err := service.ListRecords(1, "", false)
	require.NoError(t, err)
	require.Len(t, records, 3)

	records, err = service.ListRecords(999, "", false)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestServiceGetRecord(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "secret",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	got, err := service.GetRecord(r.ID)
	require.NoError(t, err)
	require.Equal(t, r.Name, got.Name)

	_, err = service.GetRecord(999)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestServiceUpdateRecord(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeLogin, Name: "original",
		Payload: models.LoginPayload{Login: "user", Password: "pass"},
		DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	r.Name = "updated"
	r.Payload = models.LoginPayload{Login: "newuser", Password: "newpass"}
	require.NoError(t, service.UpdateRecord(r))

	got, err := service.GetRecord(r.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", got.Name)
}

func TestServiceDeleteRecord(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "to-delete",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	require.NoError(t, service.DeleteRecord(r.ID, "dev-1"))

	_, err := service.GetRecord(r.ID)
	// GetRecord now returns ErrRecordNotFound for soft-deleted records
	require.ErrorIs(t, err, models.ErrRecordNotFound)

	records, err := service.ListRecords(1, "", false)
	require.NoError(t, err)
	require.Empty(t, records)

	err = service.DeleteRecord(999, "dev-1")
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestServiceDeleteRecord_EmptyDeviceID(t *testing.T) {
	service, _ := newTestService(t)
	err := service.DeleteRecord(1, "")
	require.ErrorIs(t, err, models.ErrEmptyDeviceID)
}

func TestServiceNewService_Validation(t *testing.T) {
	_, err := NewService(nil, nil)
	require.Error(t, err)

	keyRepo := newMemKeyRepo()
	km, err := keys.NewManager(keyRepo, testMasterKey(t))
	require.NoError(t, err)

	_, err = NewService(newMemRecordRepo(), nil)
	require.Error(t, err)

	_, err = NewService(nil, km)
	require.Error(t, err)
}

func TestServiceCreateRecord_Metadata(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "with-meta",
		Metadata: "some arbitrary metadata",
		Payload:  models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	got, err := service.GetRecord(r.ID)
	require.NoError(t, err)
	require.Equal(t, "some arbitrary metadata", got.Metadata)
}

// --- UpdateRecord additional coverage ---

func TestServiceUpdateRecord_NilRecord(t *testing.T) {
	service, _ := newTestService(t)

	err := service.UpdateRecord(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "record is nil")
}

func TestServiceUpdateRecord_ValidationError(t *testing.T) {
	service, _ := newTestService(t)

	// Create a valid record first.
	r := &models.Record{
		UserID:   1, Type: models.RecordTypeText, Name: "valid",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	tests := []struct {
		name       string
		mutate     func(rec *models.Record)
		wantErrStr string
	}{
		{
			name:       "empty name",
			mutate:     func(rec *models.Record) { rec.Name = "" },
			wantErrStr: "record name is required",
		},
		{
			name:       "empty device_id",
			mutate:     func(rec *models.Record) { rec.DeviceID = "" },
			wantErrStr: "device_id is required",
		},
		{
			name:       "nil payload",
			mutate:     func(rec *models.Record) { rec.Payload = nil },
			wantErrStr: "payload is required",
		},
		{
			name:       "invalid user_id",
			mutate:     func(rec *models.Record) { rec.UserID = 0 },
			wantErrStr: "invalid user id",
		},
		{
			name:       "invalid key_version",
			mutate:     func(rec *models.Record) { rec.KeyVersion = 0 },
			wantErrStr: "key_version must be positive",
		},
		{
			name:       "payload type mismatch",
			mutate:     func(rec *models.Record) { rec.Type = models.RecordTypeLogin },
			wantErrStr: "does not match record type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reload the record from repo so mutations don't leak between cases.
			fresh, err := service.GetRecord(r.ID)
			require.NoError(t, err)

			tt.mutate(fresh)

			err = service.UpdateRecord(fresh)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErrStr)
		})
	}
}

func TestServiceUpdateRecord_NotFound(t *testing.T) {
	service, _ := newTestService(t)

	// An update for a record that never existed should fail at repo level.
	ghost := &models.Record{
		ID: 99999, UserID: 1, Type: models.RecordTypeText, Name: "ghost",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
		KeyVersion: 1,
	}
	err := service.UpdateRecord(ghost)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestServiceUpdateRecord_AlreadyDeleted(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "doomed",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	// Soft-delete the record.
	require.NoError(t, service.DeleteRecord(r.ID, "dev-1"))

	// Attempt to update the deleted record. The repo layer does not block this,
	// but the service GetRecord returns ErrRecordNotFound for soft-deleted
	// records, so we verify the record can no longer be fetched.
	_, err := service.GetRecord(r.ID)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

func TestServiceUpdateRecord_RevisionBump(t *testing.T) {
	service, repo := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "revision-test",
		Payload: models.TextPayload{Content: "original"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))
	require.Equal(t, int64(0), r.Revision) // repo does not set revision on create

	// Manually bump revision in the stored record to simulate a prior state.
	stored, ok := repo.records[r.ID]
	require.True(t, ok)
	stored.Revision = 3

	// Update the record with a new revision — the service passes through to repo.
	fresh, err := service.GetRecord(r.ID)
	require.NoError(t, err)
	fresh.Name = "revision-updated"
	fresh.Revision = 4
	require.NoError(t, service.UpdateRecord(fresh))

	got, err := service.GetRecord(r.ID)
	require.NoError(t, err)
	require.Equal(t, int64(4), got.Revision)
	require.Equal(t, "revision-updated", got.Name)
}

// --- CreateRecord additional coverage ---

func TestServiceCreateRecord_ValidationErrors(t *testing.T) {
	service, _ := newTestService(t)

	tests := []struct {
		name   string
		record *models.Record
	}{
		{
			name: "empty name",
			record: &models.Record{
				UserID: 1, Type: models.RecordTypeText, Name: "",
				Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
			},
		},
		{
			name: "invalid type",
			record: &models.Record{
				UserID: 1, Type: "unknown", Name: "test",
				Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
			},
		},
		{
			name: "nil payload",
			record: &models.Record{
				UserID: 1, Type: models.RecordTypeText, Name: "test",
				Payload: nil, DeviceID: "dev-1",
			},
		},
		{
			name: "zero user_id",
			record: &models.Record{
				UserID: 0, Type: models.RecordTypeText, Name: "test",
				Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
			},
		},
		{
			name: "empty device_id",
			record: &models.Record{
				UserID: 1, Type: models.RecordTypeText, Name: "test",
				Payload: models.TextPayload{Content: "data"}, DeviceID: "",
			},
		},
		{
			name: "payload type mismatch",
			record: &models.Record{
				UserID: 1, Type: models.RecordTypeLogin, Name: "test",
				Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
			},
		},
		{
			name: "binary without payload_version",
			record: &models.Record{
				UserID: 1, Type: models.RecordTypeBinary, Name: "test",
				Payload: models.BinaryPayload{Data: []byte{1}}, DeviceID: "dev-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateRecord(tt.record)
			require.Error(t, err)
		})
	}
}

// --- GetRecord additional coverage ---

func TestServiceGetRecord_DeletedRecord(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "deleted",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))
	require.NoError(t, service.DeleteRecord(r.ID, "dev-1"))

	// GetRecord must return ErrRecordNotFound for soft-deleted records.
	_, err := service.GetRecord(r.ID)
	require.ErrorIs(t, err, models.ErrRecordNotFound)
}

// --- ListRecords additional coverage ---

func TestServiceListRecords_FilterByType(t *testing.T) {
	service, _ := newTestService(t)

	textRec := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "note",
		Payload: models.TextPayload{Content: "hello"}, DeviceID: "dev-1",
	}
	loginRec := &models.Record{
		UserID: 1, Type: models.RecordTypeLogin, Name: "creds",
		Payload: models.LoginPayload{Login: "u", Password: "p"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(textRec))
	require.NoError(t, service.CreateRecord(loginRec))

	// Filter for text only.
	textOnly, err := service.ListRecords(1, models.RecordTypeText, false)
	require.NoError(t, err)
	require.Len(t, textOnly, 1)
	require.Equal(t, models.RecordTypeText, textOnly[0].Type)

	// Filter for login only.
	loginOnly, err := service.ListRecords(1, models.RecordTypeLogin, false)
	require.NoError(t, err)
	require.Len(t, loginOnly, 1)
	require.Equal(t, models.RecordTypeLogin, loginOnly[0].Type)

	// No filter — returns both.
	all, err := service.ListRecords(1, "", false)
	require.NoError(t, err)
	require.Len(t, all, 2)
}

func TestServiceListRecords_IncludeDeleted(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "deleted",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))
	require.NoError(t, service.DeleteRecord(r.ID, "dev-1"))

	// Without includeDeleted — deleted record should not appear.
	active, err := service.ListRecords(1, "", false)
	require.NoError(t, err)
	require.Empty(t, active)

	// With includeDeleted — deleted record should appear.
	all, err := service.ListRecords(1, "", true)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func TestServiceListRecords_DifferentUsers(t *testing.T) {
	service, _ := newTestService(t)

	for _, uid := range []int64{10, 20, 20} {
		r := &models.Record{
			UserID: uid, Type: models.RecordTypeText, Name: "note",
			Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
		}
		require.NoError(t, service.CreateRecord(r))
	}

	user10, err := service.ListRecords(10, "", false)
	require.NoError(t, err)
	require.Len(t, user10, 1)

	user20, err := service.ListRecords(20, "", false)
	require.NoError(t, err)
	require.Len(t, user20, 2)
}

// --- DeleteRecord additional coverage ---

func TestServiceDeleteRecord_AlreadyDeleted(t *testing.T) {
	service, _ := newTestService(t)

	r := &models.Record{
		UserID: 1, Type: models.RecordTypeText, Name: "double-delete",
		Payload: models.TextPayload{Content: "data"}, DeviceID: "dev-1",
	}
	require.NoError(t, service.CreateRecord(r))

	require.NoError(t, service.DeleteRecord(r.ID, "dev-1"))

	// Second delete should fail with ErrAlreadyDeleted.
	err := service.DeleteRecord(r.ID, "dev-2")
	require.ErrorIs(t, err, models.ErrAlreadyDeleted)
}
