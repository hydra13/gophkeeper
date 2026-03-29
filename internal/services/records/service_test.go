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
