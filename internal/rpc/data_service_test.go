package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// mockRecordUseCase реализует RecordUseCase для тестов gRPC.
type mockRecordUseCase struct {
	createFn func(record *models.Record) error
	getFn    func(id int64) (*models.Record, error)
	listFn   func(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error)
	updateFn func(record *models.Record) error
	deleteFn func(id int64, deviceID string) error
}

func (m *mockRecordUseCase) CreateRecord(record *models.Record) error {
	return m.createFn(record)
}

func (m *mockRecordUseCase) GetRecord(id int64) (*models.Record, error) {
	return m.getFn(id)
}

func (m *mockRecordUseCase) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return m.listFn(userID, recordType, includeDeleted)
}

func (m *mockRecordUseCase) UpdateRecord(record *models.Record) error {
	return m.updateFn(record)
}

func (m *mockRecordUseCase) DeleteRecord(id int64, deviceID string) error {
	return m.deleteFn(id, deviceID)
}

func newTestDataService(mock *mockRecordUseCase) *DataService {
	return NewDataService(mock, zerolog.Nop())
}

func ctxWithUser(userID int64) context.Context {
	return middlewares.ContextWithUserID(context.Background(), userID)
}

func sampleRecord() *models.Record {
	return &models.Record{
		ID:             1,
		UserID:         10,
		Type:           models.RecordTypeLogin,
		Name:           "my login",
		Metadata:       "some metadata",
		Payload:        models.LoginPayload{Login: "user", Password: "pass"},
		Revision:       1,
		DeviceID:       "dev-1",
		KeyVersion:     1,
		PayloadVersion: 0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// --- CreateRecord ---

func TestDataService_CreateRecord_Success(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			record.ID = 42
			return nil
		},
	}
	svc := newTestDataService(mock)

	resp, err := svc.CreateRecord(ctxWithUser(10), &pbv1.CreateRecordRequest{
		Type:     pbv1.RecordType_RECORD_TYPE_LOGIN,
		Name:     "my login",
		Metadata: "meta",
		DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Login{
			Login: &pbv1.LoginPayload{Login: "user", Password: "pass"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
	require.Equal(t, int64(42), resp.Record.Id)
	require.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, resp.Record.Type)
}

func TestDataService_CreateRecord_LoginType(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			record.ID = 1
			return nil
		},
	}
	svc := newTestDataService(mock)

	resp, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Type: pbv1.RecordType_RECORD_TYPE_LOGIN, Name: "test", DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
}

func TestDataService_CreateRecord_TextType(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			record.ID = 1
			return nil
		},
	}
	svc := newTestDataService(mock)

	resp, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Type: pbv1.RecordType_RECORD_TYPE_TEXT, Name: "test", DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Text{Text: &pbv1.TextPayload{Content: "hello"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
}

func TestDataService_CreateRecord_BinaryType(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			record.ID = 1
			return nil
		},
	}
	svc := newTestDataService(mock)

	resp, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Type: pbv1.RecordType_RECORD_TYPE_BINARY, Name: "test", DeviceId: "dev-1",
		Payload:        &pbv1.CreateRecordRequest_Binary{Binary: &pbv1.BinaryPayload{}},
		PayloadVersion: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
}

func TestDataService_CreateRecord_CardType(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			record.ID = 1
			return nil
		},
	}
	svc := newTestDataService(mock)

	resp, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Type: pbv1.RecordType_RECORD_TYPE_CARD, Name: "test", DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Card{Card: &pbv1.CardPayload{Number: "4111", HolderName: "T", ExpiryDate: "12/25", Cvv: "123"}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
}

func TestDataService_CreateRecord_NoAuth(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.CreateRecord(context.Background(), &pbv1.CreateRecordRequest{
		Name: "test", DeviceId: "dev",
		Payload: &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}},
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestDataService_CreateRecord_MissingName(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDataService_CreateRecord_MissingDeviceID(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Name: "test",
		Payload: &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDataService_CreateRecord_NoPayload(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Name: "test", DeviceId: "dev-1",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDataService_CreateRecord_ValidationError(t *testing.T) {
	mock := &mockRecordUseCase{
		createFn: func(record *models.Record) error {
			return models.ErrEmptyRecordName
		},
	}
	svc := newTestDataService(mock)
	_, err := svc.CreateRecord(ctxWithUser(1), &pbv1.CreateRecordRequest{
		Name: "test", DeviceId: "dev-1",
		Payload: &pbv1.CreateRecordRequest_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// --- GetRecord ---

func TestDataService_GetRecord_Success(t *testing.T) {
	rec := sampleRecord()
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return rec, nil }}
	svc := newTestDataService(mock)

	resp, err := svc.GetRecord(ctxWithUser(10), &pbv1.GetRecordRequest{Id: 1})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Record.Id)
	require.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, resp.Record.Type)
}

func TestDataService_GetRecord_NotFound(t *testing.T) {
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return nil, models.ErrRecordNotFound }}
	svc := newTestDataService(mock)

	_, err := svc.GetRecord(ctxWithUser(10), &pbv1.GetRecordRequest{Id: 999})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestDataService_GetRecord_AccessDenied(t *testing.T) {
	rec := sampleRecord() // UserID=10
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return rec, nil }}
	svc := newTestDataService(mock)

	_, err := svc.GetRecord(ctxWithUser(99), &pbv1.GetRecordRequest{Id: 1})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestDataService_GetRecord_InvalidID(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.GetRecord(ctxWithUser(1), &pbv1.GetRecordRequest{Id: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// --- ListRecords ---

func TestDataService_ListRecords_Success(t *testing.T) {
	records := []models.Record{*sampleRecord()}
	mock := &mockRecordUseCase{listFn: func(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) { return records, nil }}
	svc := newTestDataService(mock)

	resp, err := svc.ListRecords(ctxWithUser(10), &pbv1.ListRecordsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Records, 1)
	require.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, resp.Records[0].Type)
}

func TestDataService_ListRecords_Empty(t *testing.T) {
	mock := &mockRecordUseCase{listFn: func(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) { return nil, nil }}
	svc := newTestDataService(mock)

	resp, err := svc.ListRecords(ctxWithUser(10), &pbv1.ListRecordsRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.Records)
}

func TestDataService_ListRecords_NoAuth(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.ListRecords(context.Background(), &pbv1.ListRecordsRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// --- UpdateRecord ---

func TestDataService_UpdateRecord_Success(t *testing.T) {
	rec := sampleRecord()
	mock := &mockRecordUseCase{
		getFn:    func(id int64) (*models.Record, error) { return rec, nil },
		updateFn: func(record *models.Record) error { return nil },
	}
	svc := newTestDataService(mock)

	resp, err := svc.UpdateRecord(ctxWithUser(10), &pbv1.UpdateRecordRequest{
		Id:       1,
		Name:     "updated",
		Metadata: "new meta",
		DeviceId: "dev-2",
		Revision: 2,
		Payload: &pbv1.UpdateRecordRequest_Login{
			Login: &pbv1.LoginPayload{Login: "newuser", Password: "newpass"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
	require.Equal(t, int64(2), resp.Record.Revision)
}

func TestDataService_UpdateRecord_NotFound(t *testing.T) {
	mock := &mockRecordUseCase{
		getFn: func(id int64) (*models.Record, error) { return nil, models.ErrRecordNotFound },
	}
	svc := newTestDataService(mock)

	_, err := svc.UpdateRecord(ctxWithUser(10), &pbv1.UpdateRecordRequest{
		Id: 999, Name: "x", DeviceId: "d", Revision: 2,
		Payload: &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{}},
	})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestDataService_UpdateRecord_AccessDenied(t *testing.T) {
	rec := sampleRecord() // UserID=10
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return rec, nil }}
	svc := newTestDataService(mock)

	_, err := svc.UpdateRecord(ctxWithUser(99), &pbv1.UpdateRecordRequest{
		Id: 1, Name: "x", DeviceId: "d", Revision: 2,
		Payload: &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{}},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestDataService_UpdateRecord_Deleted(t *testing.T) {
	rec := sampleRecord()
	now := time.Now()
	rec.DeletedAt = &now
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return rec, nil }}
	svc := newTestDataService(mock)

	_, err := svc.UpdateRecord(ctxWithUser(10), &pbv1.UpdateRecordRequest{
		Id: 1, Name: "x", DeviceId: "d", Revision: 2,
		Payload: &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{}},
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestDataService_UpdateRecord_RevisionConflict(t *testing.T) {
	rec := sampleRecord() // Revision=1
	mock := &mockRecordUseCase{getFn: func(id int64) (*models.Record, error) { return rec, nil }}
	svc := newTestDataService(mock)

	_, err := svc.UpdateRecord(ctxWithUser(10), &pbv1.UpdateRecordRequest{
		Id: 1, Name: "x", DeviceId: "d", Revision: 1,
		Payload: &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{}},
	})
	require.Equal(t, codes.Aborted, status.Code(err))
}

func TestDataService_UpdateRecord_MissingDeviceID(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.UpdateRecord(ctxWithUser(1), &pbv1.UpdateRecordRequest{
		Id: 1, Name: "x", Revision: 2,
		Payload: &pbv1.UpdateRecordRequest_Login{Login: &pbv1.LoginPayload{}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// --- DeleteRecord ---

func TestDataService_DeleteRecord_Success(t *testing.T) {
	rec := sampleRecord()
	mock := &mockRecordUseCase{
		getFn:    func(id int64) (*models.Record, error) { return rec, nil },
		deleteFn: func(id int64, deviceID string) error { return nil },
	}
	svc := newTestDataService(mock)

	resp, err := svc.DeleteRecord(ctxWithUser(10), &pbv1.DeleteRecordRequest{
		Id: 1, DeviceId: "dev-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestDataService_DeleteRecord_NotFound(t *testing.T) {
	mock := &mockRecordUseCase{
		getFn: func(id int64) (*models.Record, error) { return nil, models.ErrRecordNotFound },
	}
	svc := newTestDataService(mock)

	_, err := svc.DeleteRecord(ctxWithUser(10), &pbv1.DeleteRecordRequest{Id: 999, DeviceId: "d"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestDataService_DeleteRecord_AccessDenied(t *testing.T) {
	rec := sampleRecord() // UserID=10
	mock := &mockRecordUseCase{
		getFn: func(id int64) (*models.Record, error) { return rec, nil },
	}
	svc := newTestDataService(mock)

	_, err := svc.DeleteRecord(ctxWithUser(99), &pbv1.DeleteRecordRequest{Id: 1, DeviceId: "d"})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestDataService_DeleteRecord_AlreadyDeleted(t *testing.T) {
	rec := sampleRecord()
	now := time.Now()
	rec.DeletedAt = &now
	mock := &mockRecordUseCase{
		getFn: func(id int64) (*models.Record, error) { return rec, nil },
	}
	svc := newTestDataService(mock)

	_, err := svc.DeleteRecord(ctxWithUser(10), &pbv1.DeleteRecordRequest{Id: 1, DeviceId: "d"})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestDataService_DeleteRecord_MissingDeviceID(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.DeleteRecord(ctxWithUser(1), &pbv1.DeleteRecordRequest{Id: 1})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDataService_DeleteRecord_InvalidID(t *testing.T) {
	svc := newTestDataService(&mockRecordUseCase{})
	_, err := svc.DeleteRecord(ctxWithUser(1), &pbv1.DeleteRecordRequest{Id: 0, DeviceId: "d"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// --- domainRecordToProto ---

func TestDomainRecordToProto_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		payload  models.RecordPayload
		wantType pbv1.RecordType
	}{
		{"login", models.LoginPayload{Login: "u", Password: "p"}, pbv1.RecordType_RECORD_TYPE_LOGIN},
		{"text", models.TextPayload{Content: "hello"}, pbv1.RecordType_RECORD_TYPE_TEXT},
		{"binary", models.BinaryPayload{Data: []byte{1, 2}}, pbv1.RecordType_RECORD_TYPE_BINARY},
		{"card", models.CardPayload{Number: "4111", HolderName: "T", ExpiryDate: "12/25", CVV: "123"}, pbv1.RecordType_RECORD_TYPE_CARD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &models.Record{
				ID: 1, UserID: 10, Type: models.RecordType(tt.name),
				Name: "test", Payload: tt.payload,
				KeyVersion: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			pb := domainRecordToProto(r)
			require.Equal(t, tt.wantType, pb.Type)
		})
	}
}

func TestDomainRecordToProto_Nil(t *testing.T) {
	require.Nil(t, domainRecordToProto(nil))
}

func TestDomainRecordToProto_DeletedAt(t *testing.T) {
	now := time.Now()
	r := &models.Record{
		ID: 1, Type: models.RecordTypeText, Payload: models.TextPayload{Content: "x"},
		KeyVersion: 1, DeletedAt: &now, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	pb := domainRecordToProto(r)
	require.NotNil(t, pb.DeletedAt)
}

// --- mapRecordError ---

func TestMapRecordError(t *testing.T) {
	tests := []struct {
		err  error
		want codes.Code
	}{
		{models.ErrRecordNotFound, codes.NotFound},
		{models.ErrAlreadyDeleted, codes.FailedPrecondition},
		{models.ErrRevisionConflict, codes.Aborted},
		{models.ErrEmptyRecordName, codes.InvalidArgument},
		{models.ErrEmptyDeviceID, codes.InvalidArgument},
		{models.ErrNilPayload, codes.InvalidArgument},
		{models.ErrInvalidKeyVersion, codes.InvalidArgument},
		{models.ErrInvalidUserID, codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			got := mapRecordError(tt.err)
			require.Equal(t, tt.want, status.Code(got))
		})
	}
}
