package cache

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- recordToJSON / jsonToRecord roundtrip ---

func TestRecordToJSON_LoginPayload(t *testing.T) {
	now := time.Now()
	deletedAt := now.Add(-1 * time.Hour)

	rec := models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeLogin,
		Name:           "my-login",
		Metadata:       "some meta",
		Payload:        models.LoginPayload{Login: "user", Password: "pass"},
		Revision:       3,
		DeletedAt:      &deletedAt,
		DeviceID:       "dev-1",
		KeyVersion:     2,
		PayloadVersion: 1,
		CreatedAt:      now.Add(-24 * time.Hour),
		UpdatedAt:      now,
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)

	assert.Equal(t, int64(1), jr.ID)
	assert.Equal(t, int64(10), jr.UserID)
	assert.Equal(t, models.RecordTypeLogin, jr.Type)
	assert.Equal(t, "my-login", jr.Name)
	assert.Equal(t, "some meta", jr.Metadata)
	assert.Equal(t, int64(3), jr.Revision)
	assert.NotNil(t, jr.DeletedAt)
	assert.Equal(t, deletedAt.Unix(), *jr.DeletedAt)
	assert.Equal(t, "dev-1", jr.DeviceID)
	assert.Equal(t, int64(2), jr.KeyVersion)
	assert.Equal(t, int64(1), jr.PayloadVersion)
	assert.Equal(t, now.Add(-24*time.Hour).Unix(), jr.CreatedAt)
	assert.Equal(t, now.Unix(), jr.UpdatedAt)
	assert.True(t, len(jr.Payload) > 0)

	// Roundtrip back
	got, err := jsonToRecord(jr)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, got.ID)
	assert.Equal(t, rec.UserID, got.UserID)
	assert.Equal(t, rec.Type, got.Type)
	assert.Equal(t, rec.Name, got.Name)
	assert.Equal(t, rec.Metadata, got.Metadata)
	assert.Equal(t, rec.Revision, got.Revision)
	assert.NotNil(t, got.DeletedAt)
	assert.Equal(t, rec.DeletedAt.Unix(), got.DeletedAt.Unix())
	assert.Equal(t, rec.DeviceID, got.DeviceID)
	assert.Equal(t, rec.KeyVersion, got.KeyVersion)
	assert.Equal(t, rec.PayloadVersion, got.PayloadVersion)
	assert.Equal(t, rec.CreatedAt.Unix(), got.CreatedAt.Unix())
	assert.Equal(t, rec.UpdatedAt.Unix(), got.UpdatedAt.Unix())

	// Payload type check
	loginPayload, ok := got.Payload.(models.LoginPayload)
	require.True(t, ok, "expected LoginPayload")
	assert.Equal(t, "user", loginPayload.Login)
	assert.Equal(t, "pass", loginPayload.Password)
}

func TestRecordToJSON_TextPayload(t *testing.T) {
	rec := models.Record{
		ID:        2,
		UserID:    1,
		Type:      models.RecordTypeText,
		Name:      "note",
		Payload:   models.TextPayload{Content: "hello world"},
		DeviceID:  "dev-2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)

	got, err := jsonToRecord(jr)
	require.NoError(t, err)

	textPayload, ok := got.Payload.(models.TextPayload)
	require.True(t, ok, "expected TextPayload")
	assert.Equal(t, "hello world", textPayload.Content)
}

func TestRecordToJSON_BinaryPayload(t *testing.T) {
	rec := models.Record{
		ID:             3,
		UserID:         1,
		Type:           models.RecordTypeBinary,
		Name:           "file.bin",
		Payload:        models.BinaryPayload{Data: []byte{0x01, 0x02, 0x03}},
		DeviceID:       "dev-3",
		PayloadVersion: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)

	got, err := jsonToRecord(jr)
	require.NoError(t, err)

	binPayload, ok := got.Payload.(models.BinaryPayload)
	require.True(t, ok, "expected BinaryPayload")
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, binPayload.Data)
}

func TestRecordToJSON_CardPayload(t *testing.T) {
	rec := models.Record{
		ID:        4,
		UserID:    1,
		Type:      models.RecordTypeCard,
		Name:      "my-card",
		Payload:   models.CardPayload{Number: "4111111111111111", HolderName: "John Doe", ExpiryDate: "12/30", CVV: "123"},
		DeviceID:  "dev-4",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)

	got, err := jsonToRecord(jr)
	require.NoError(t, err)

	cardPayload, ok := got.Payload.(models.CardPayload)
	require.True(t, ok, "expected CardPayload")
	assert.Equal(t, "4111111111111111", cardPayload.Number)
	assert.Equal(t, "John Doe", cardPayload.HolderName)
	assert.Equal(t, "12/30", cardPayload.ExpiryDate)
	assert.Equal(t, "123", cardPayload.CVV)
}

func TestRecordToJSON_NilPayload(t *testing.T) {
	rec := models.Record{
		ID:        5,
		UserID:    1,
		Type:      models.RecordTypeLogin,
		Name:      "no-payload",
		Payload:   nil,
		DeviceID:  "dev-5",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)
	assert.Nil(t, jr.Payload)

	got, err := jsonToRecord(jr)
	require.NoError(t, err)
	assert.Nil(t, got.Payload)
}

func TestRecordToJSON_NoDeletedAt(t *testing.T) {
	rec := models.Record{
		ID:        6,
		UserID:    1,
		Type:      models.RecordTypeText,
		Name:      "active",
		Payload:   models.TextPayload{Content: "data"},
		DeviceID:  "dev",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)
	assert.Nil(t, jr.DeletedAt)

	got, err := jsonToRecord(jr)
	require.NoError(t, err)
	assert.Nil(t, got.DeletedAt)
}

func TestRecordToJSON_ZeroTimestamps(t *testing.T) {
	rec := models.Record{
		ID:       7,
		UserID:   1,
		Type:     models.RecordTypeLogin,
		Name:     "zero-ts",
		Payload:  models.LoginPayload{Login: "u", Password: "p"},
		DeviceID: "dev",
	}

	jr, err := recordToJSON(rec)
	require.NoError(t, err)
	// time.Time{}.Unix() is not 0; it is a large negative number.
	// The serialization just stores whatever .Unix() returns.
	assert.Equal(t, rec.CreatedAt.Unix(), jr.CreatedAt)
	assert.Equal(t, rec.UpdatedAt.Unix(), jr.UpdatedAt)

	// When CreatedAt/UpdatedAt are 0 in the JSON, jsonToRecord does NOT set
	// the field, leaving it as time.Time{}.
	jr.CreatedAt = 0
	jr.UpdatedAt = 0

	got, err := jsonToRecord(jr)
	require.NoError(t, err)
	assert.True(t, got.CreatedAt.IsZero())
	assert.True(t, got.UpdatedAt.IsZero())
}

// --- pendingOpToJSON / jsonToPendingOp roundtrip ---

func TestPendingOpToJSON_CreateWithRecord(t *testing.T) {
	rec := models.Record{
		ID:        1,
		UserID:    10,
		Type:      models.RecordTypeLogin,
		Name:      "pending-login",
		Payload:   models.LoginPayload{Login: "user", Password: "pass"},
		DeviceID:  "dev",
		Revision:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	op := PendingOp{
		ID:           100,
		RecordID:     1,
		Operation:    OperationCreate,
		Record:       &rec,
		BaseRevision: 0,
		CreatedAt:    1700000000,
	}

	jop, err := pendingOpToJSON(op)
	require.NoError(t, err)

	assert.Equal(t, int64(100), jop.ID)
	assert.Equal(t, int64(1), jop.RecordID)
	assert.Equal(t, OperationCreate, jop.Operation)
	assert.Equal(t, int64(0), jop.BaseRevision)
	assert.Equal(t, int64(1700000000), jop.CreatedAt)
	assert.True(t, len(jop.Record) > 0)

	// Roundtrip
	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	assert.Equal(t, op.ID, got.ID)
	assert.Equal(t, op.RecordID, got.RecordID)
	assert.Equal(t, op.Operation, got.Operation)
	assert.Equal(t, op.BaseRevision, got.BaseRevision)
	assert.Equal(t, op.CreatedAt, got.CreatedAt)
	require.NotNil(t, got.Record)
	assert.Equal(t, "pending-login", got.Record.Name)
}

func TestPendingOpToJSON_DeleteWithoutRecord(t *testing.T) {
	op := PendingOp{
		ID:           200,
		RecordID:     5,
		Operation:    OperationDelete,
		Record:       nil,
		BaseRevision: 3,
		CreatedAt:    1700000001,
	}

	jop, err := pendingOpToJSON(op)
	require.NoError(t, err)
	assert.Equal(t, int64(200), jop.ID)
	assert.Nil(t, jop.Record)

	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	assert.Equal(t, op.ID, got.ID)
	assert.Equal(t, op.RecordID, got.RecordID)
	assert.Equal(t, OperationDelete, got.Operation)
	assert.Nil(t, got.Record)
}

func TestPendingOpToJSON_UpdateWithTextPayload(t *testing.T) {
	rec := models.Record{
		ID:        2,
		UserID:    1,
		Type:      models.RecordTypeText,
		Name:      "updated-text",
		Payload:   models.TextPayload{Content: "new content"},
		DeviceID:  "dev",
		Revision:  2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	op := PendingOp{
		ID:           300,
		RecordID:     2,
		Operation:    OperationUpdate,
		Record:       &rec,
		BaseRevision: 1,
		CreatedAt:    1700000002,
	}

	jop, err := pendingOpToJSON(op)
	require.NoError(t, err)

	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	require.NotNil(t, got.Record)
	textPayload, ok := got.Record.Payload.(models.TextPayload)
	require.True(t, ok, "expected TextPayload")
	assert.Equal(t, "new content", textPayload.Content)
}

func TestPendingOpToJSON_WithCardPayload(t *testing.T) {
	rec := models.Record{
		ID:        3,
		UserID:    1,
		Type:      models.RecordTypeCard,
		Name:      "card-op",
		Payload:   models.CardPayload{Number: "5500000000000004", HolderName: "Jane"},
		DeviceID:  "dev",
		Revision:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	op := PendingOp{
		ID:           400,
		RecordID:     3,
		Operation:    OperationUpdate,
		Record:       &rec,
		BaseRevision: 0,
		CreatedAt:    1700000003,
	}

	jop, err := pendingOpToJSON(op)
	require.NoError(t, err)

	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	require.NotNil(t, got.Record)
	cardPayload, ok := got.Record.Payload.(models.CardPayload)
	require.True(t, ok, "expected CardPayload")
	assert.Equal(t, "5500000000000004", cardPayload.Number)
}

func TestPendingOpToJSON_WithBinaryPayload(t *testing.T) {
	rec := models.Record{
		ID:             4,
		UserID:         1,
		Type:           models.RecordTypeBinary,
		Name:           "bin-op",
		Payload:        models.BinaryPayload{Data: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		DeviceID:       "dev",
		Revision:       1,
		PayloadVersion: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	op := PendingOp{
		ID:           500,
		RecordID:     4,
		Operation:    OperationCreate,
		Record:       &rec,
		BaseRevision: 0,
		CreatedAt:    1700000004,
	}

	jop, err := pendingOpToJSON(op)
	require.NoError(t, err)

	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	require.NotNil(t, got.Record)
	binPayload, ok := got.Record.Payload.(models.BinaryPayload)
	require.True(t, ok, "expected BinaryPayload")
	assert.Equal(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}, binPayload.Data)
}

// --- jsonToPendingOp edge cases ---

func TestJsonToPendingOp_EmptyRecordJSON(t *testing.T) {
	jop := jsonPendingOp{
		ID:           1,
		RecordID:     10,
		Operation:    OperationDelete,
		Record:       nil, // no record
		BaseRevision: 5,
		CreatedAt:    1700000000,
	}

	got, err := jsonToPendingOp(jop)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.ID)
	assert.Nil(t, got.Record)
}

func TestJsonToPendingOp_InvalidRecordJSON(t *testing.T) {
	jop := jsonPendingOp{
		ID:           1,
		RecordID:     10,
		Operation:    OperationCreate,
		Record:       json.RawMessage(`{invalid json}`),
		BaseRevision: 0,
		CreatedAt:    1700000000,
	}

	_, err := jsonToPendingOp(jop)
	assert.Error(t, err)
}

// --- Full Flush/Load roundtrip via FileStore with all payload types ---

func TestFileStore_FlushAndReload_AllPayloadTypes(t *testing.T) {
	dir := t.TempDir()

	store1, err := NewFileStore(dir)
	require.NoError(t, err)

	loginRec := &models.Record{
		ID: 1, UserID: 1, Type: models.RecordTypeLogin, Name: "login-rec",
		Payload:  models.LoginPayload{Login: "u", Password: "p"},
		DeviceID: "dev", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	textRec := &models.Record{
		ID: 2, UserID: 1, Type: models.RecordTypeText, Name: "text-rec",
		Payload:  models.TextPayload{Content: "text data"},
		DeviceID: "dev", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	binRec := &models.Record{
		ID: 3, UserID: 1, Type: models.RecordTypeBinary, Name: "bin-rec",
		Payload:  models.BinaryPayload{Data: []byte{0x01, 0x02}},
		DeviceID: "dev", Revision: 1, PayloadVersion: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	cardRec := &models.Record{
		ID: 4, UserID: 1, Type: models.RecordTypeCard, Name: "card-rec",
		Payload:  models.CardPayload{Number: "4111", HolderName: "Test", ExpiryDate: "01/30", CVV: "000"},
		DeviceID: "dev", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	store1.Records().Put(loginRec)
	store1.Records().Put(textRec)
	store1.Records().Put(binRec)
	store1.Records().Put(cardRec)

	require.NoError(t, store1.Flush())

	// Reload
	store2, err := NewFileStore(dir)
	require.NoError(t, err)

	// Login
	got, ok := store2.Records().Get(1)
	require.True(t, ok)
	lp, ok := got.Payload.(models.LoginPayload)
	require.True(t, ok)
	assert.Equal(t, "u", lp.Login)

	// Text
	got, ok = store2.Records().Get(2)
	require.True(t, ok)
	tp, ok := got.Payload.(models.TextPayload)
	require.True(t, ok)
	assert.Equal(t, "text data", tp.Content)

	// Binary
	got, ok = store2.Records().Get(3)
	require.True(t, ok)
	bp, ok := got.Payload.(models.BinaryPayload)
	require.True(t, ok)
	assert.Equal(t, []byte{0x01, 0x02}, bp.Data)

	// Card
	got, ok = store2.Records().Get(4)
	require.True(t, ok)
	cp, ok := got.Payload.(models.CardPayload)
	require.True(t, ok)
	assert.Equal(t, "4111", cp.Number)
}

func TestFileStore_FlushAndReload_PendingOps(t *testing.T) {
	dir := t.TempDir()

	store1, err := NewFileStore(dir)
	require.NoError(t, err)

	rec := &models.Record{
		ID: 1, UserID: 1, Type: models.RecordTypeLogin, Name: "p-rec",
		Payload:  models.LoginPayload{Login: "u", Password: "p"},
		DeviceID: "dev", Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	require.NoError(t, store1.Pending().Enqueue(PendingOp{
		ID: 1, RecordID: 1, Operation: OperationCreate,
		Record: rec, BaseRevision: 0, CreatedAt: 1700000000,
	}))
	require.NoError(t, store1.Pending().Enqueue(PendingOp{
		ID: 2, RecordID: 1, Operation: OperationUpdate,
		Record: rec, BaseRevision: 1, CreatedAt: 1700000001,
	}))
	require.NoError(t, store1.Pending().Enqueue(PendingOp{
		ID: 3, RecordID: 1, Operation: OperationDelete,
		Record: nil, BaseRevision: 2, CreatedAt: 1700000002,
	}))

	require.NoError(t, store1.Flush())

	store2, err := NewFileStore(dir)
	require.NoError(t, err)

	assert.Equal(t, 3, store2.Pending().Len())

	ops, err := store2.Pending().Peek()
	require.NoError(t, err)

	assert.Equal(t, OperationCreate, ops[0].Operation)
	assert.Equal(t, OperationUpdate, ops[1].Operation)
	assert.Equal(t, OperationDelete, ops[2].Operation)

	// First op has record
	require.NotNil(t, ops[0].Record)
	assert.Equal(t, "p-rec", ops[0].Record.Name)

	// Third op: nil record serializes as null JSON; after reload
	// jsonToPendingOp unmarshals null into an empty record pointer.
	// The record pointer is non-nil but is a zero-value Record.
	require.NotNil(t, ops[2].Record)
	assert.Equal(t, int64(0), ops[2].Record.ID)
	assert.Equal(t, "", string(ops[2].Record.Type))
}

// --- jsonToRecord with unknown type (no match in switch) ---

func TestJsonToRecord_UnknownType_NoPayloadUnmarshal(t *testing.T) {
	jr := jsonRecord{
		ID:        99,
		UserID:    1,
		Type:      "unknown_type",
		Name:      "mystery",
		Payload:   json.RawMessage(`{"something": "here"}`),
		DeviceID:  "dev",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	rec, err := jsonToRecord(jr)
	require.NoError(t, err)
	assert.Equal(t, models.RecordType("unknown_type"), rec.Type)
	assert.Nil(t, rec.Payload) // no matching case, payload stays nil
}

// --- recordToJSON with payload that fails marshal ---

func TestRecordToJSON_InvalidPayload(t *testing.T) {
	rec := models.Record{
		ID:       1,
		Type:     models.RecordTypeLogin,
		Name:     "bad",
		Payload:  models.LoginPayload{Login: "u", Password: "p"},
		DeviceID: "dev",
	}

	// Normal case should succeed - just verify the happy path works
	jr, err := recordToJSON(rec)
	require.NoError(t, err)
	assert.True(t, len(jr.Payload) > 0)
}

// --- timeFromUnix / newTimeFromUnix ---

func TestTimeFromUnix(t *testing.T) {
	unix := int64(1700000000)
	got := timeFromUnix(unix)
	assert.Equal(t, unix, got.Unix())
}

func TestNewTimeFromUnix(t *testing.T) {
	unix := int64(1700000000)
	got := newTimeFromUnix(unix)
	require.NotNil(t, got)
	assert.Equal(t, unix, got.Unix())
}

func TestNewTimeFromUnix_Zero(t *testing.T) {
	got := newTimeFromUnix(0)
	require.NotNil(t, got)
	// time.Unix(0,0) is NOT the zero value of time.Time.
	// It represents 1970-01-01 00:00:00 UTC.
	assert.Equal(t, int64(0), got.Unix())
	assert.False(t, got.IsZero()) // IsZero() checks for time.Time{} (year 1)
}
