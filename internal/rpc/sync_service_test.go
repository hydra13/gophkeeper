package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// --- mock ---

type mockSyncUseCase struct {
	pushFn           func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
	pullFn           func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
	getConflictsFn   func(userID int64) ([]models.SyncConflict, error)
	resolveConflictFn func(userID int64, conflictID int64, resolution string) (*models.Record, error)
}

func (m *mockSyncUseCase) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	return m.pushFn(userID, deviceID, changes)
}

func (m *mockSyncUseCase) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return m.pullFn(userID, deviceID, sinceRevision, limit)
}

func (m *mockSyncUseCase) GetConflicts(userID int64) ([]models.SyncConflict, error) {
	return m.getConflictsFn(userID)
}

func (m *mockSyncUseCase) ResolveConflict(userID int64, conflictID int64, resolution string) (*models.Record, error) {
	return m.resolveConflictFn(userID, conflictID, resolution)
}

func newTestSyncService(mock *mockSyncUseCase) *SyncService {
	return NewSyncService(mock, zerolog.Nop())
}

// --- helpers ---

func sampleProtoRecord() *pbv1.Record {
	return &pbv1.Record{
		Id: 1, UserId: 10, Type: pbv1.RecordType_RECORD_TYPE_LOGIN,
		Name: "test", Metadata: "meta", Revision: 1, DeviceId: "dev-1",
		KeyVersion: 1, PayloadVersion: 0,
		Payload: &pbv1.Record_Login{
			Login: &pbv1.LoginPayload{Login: "user", Password: "pass"},
		},
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}
}

func sampleDomainRecord() *models.Record {
	return &models.Record{
		ID: 1, UserID: 10, Type: models.RecordTypeLogin,
		Name: "test", Metadata: "meta",
		Payload: models.LoginPayload{Login: "user", Password: "pass"},
		Revision: 2, DeviceID: "dev-1", KeyVersion: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// --- Push ---

func TestPush_Success(t *testing.T) {
	accepted := []models.RecordRevision{
		{ID: 1, RecordID: 100, UserID: 10, Revision: 3, DeviceID: "dev-1"},
	}
	mock := &mockSyncUseCase{
		pushFn: func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return accepted, nil, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.Push(ctxWithUser(10), &pbv1.PushRequest{
		DeviceId: "dev-1",
		Changes: []*pbv1.PendingChange{
			{Record: sampleProtoRecord(), Deleted: false, BaseRevision: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Accepted, 1)
	require.Equal(t, int64(100), resp.Accepted[0].RecordId)
	require.Empty(t, resp.Conflicts)
}

func TestPush_NoAuth(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Push(context.Background(), &pbv1.PushRequest{
		DeviceId: "dev-1",
		Changes:  []*pbv1.PendingChange{{Record: sampleProtoRecord()}},
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestPush_EmptyDeviceID(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Push(ctxWithUser(1), &pbv1.PushRequest{
		DeviceId: "",
		Changes:  []*pbv1.PendingChange{{Record: sampleProtoRecord()}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPush_EmptyChanges(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Push(ctxWithUser(1), &pbv1.PushRequest{
		DeviceId: "dev-1",
		Changes:  nil,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPush_NilRecordInChange(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Push(ctxWithUser(1), &pbv1.PushRequest{
		DeviceId: "dev-1",
		Changes:  []*pbv1.PendingChange{{Record: nil}},
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPush_UseCaseError(t *testing.T) {
	mock := &mockSyncUseCase{
		pushFn: func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return nil, nil, models.ErrRevisionConflict
		},
	}
	svc := newTestSyncService(mock)

	_, err := svc.Push(ctxWithUser(1), &pbv1.PushRequest{
		DeviceId: "dev-1",
		Changes:  []*pbv1.PendingChange{{Record: sampleProtoRecord()}},
	})
	require.Equal(t, codes.Aborted, status.Code(err))
}

// --- Pull ---

func TestPull_Success(t *testing.T) {
	revisions := []models.RecordRevision{
		{ID: 1, RecordID: 100, UserID: 10, Revision: 5, DeviceID: "dev-1"},
	}
	records := []models.Record{*sampleDomainRecord()}
	mock := &mockSyncUseCase{
		pullFn: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			return revisions, records, nil, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.Pull(ctxWithUser(10), &pbv1.PullRequest{
		DeviceId:      "dev-1",
		SinceRevision: 3,
		Limit:         50,
	})
	require.NoError(t, err)
	require.Len(t, resp.Changes, 1)
	require.Equal(t, int64(100), resp.Changes[0].RecordId)
	require.Len(t, resp.Records, 1)
	require.False(t, resp.HasMore)
	require.Equal(t, int64(5), resp.NextRevision)
}

func TestPull_NoAuth(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Pull(context.Background(), &pbv1.PullRequest{DeviceId: "dev-1"})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestPull_EmptyDeviceID(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.Pull(ctxWithUser(1), &pbv1.PullRequest{DeviceId: ""})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPull_DefaultLimit(t *testing.T) {
	var capturedLimit int64
	mock := &mockSyncUseCase{
		pullFn: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			capturedLimit = limit
			return nil, nil, nil, nil
		},
	}
	svc := newTestSyncService(mock)

	_, err := svc.Pull(ctxWithUser(1), &pbv1.PullRequest{
		DeviceId: "dev-1",
		Limit:    0,
	})
	require.NoError(t, err)
	require.Equal(t, int64(50), capturedLimit)
}

func TestPull_HasMore(t *testing.T) {
	limit := int64(10)
	revisions := make([]models.RecordRevision, limit)
	for i := range revisions {
		revisions[i] = models.RecordRevision{ID: int64(i) + 1, Revision: int64(i) + 1}
	}
	mock := &mockSyncUseCase{
		pullFn: func(userID int64, deviceID string, sinceRevision int64, l int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			return revisions, nil, nil, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.Pull(ctxWithUser(1), &pbv1.PullRequest{
		DeviceId: "dev-1",
		Limit:    int32(limit),
	})
	require.NoError(t, err)
	require.True(t, resp.HasMore)
}

// --- GetConflicts ---

func TestGetConflicts_Success(t *testing.T) {
	conflicts := []models.SyncConflict{
		{
			ID: 1, UserID: 10, RecordID: 100,
			LocalRevision: 3, ServerRevision: 5,
			Resolved: false,
			LocalRecord:  sampleDomainRecord(),
			ServerRecord: sampleDomainRecord(),
		},
	}
	mock := &mockSyncUseCase{
		getConflictsFn: func(userID int64) ([]models.SyncConflict, error) {
			return conflicts, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.GetConflicts(ctxWithUser(10), &pbv1.GetConflictsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Conflicts, 1)
	require.Equal(t, int64(100), resp.Conflicts[0].RecordId)
	require.False(t, resp.Conflicts[0].Resolved)
}

func TestGetConflicts_NoAuth(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.GetConflicts(context.Background(), &pbv1.GetConflictsRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGetConflicts_UseCaseError(t *testing.T) {
	mock := &mockSyncUseCase{
		getConflictsFn: func(userID int64) ([]models.SyncConflict, error) {
			return nil, models.ErrRecordNotFound
		},
	}
	svc := newTestSyncService(mock)

	_, err := svc.GetConflicts(ctxWithUser(1), &pbv1.GetConflictsRequest{})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// --- ResolveConflict ---

func TestResolveConflict_SuccessLocal(t *testing.T) {
	record := sampleDomainRecord()
	mock := &mockSyncUseCase{
		resolveConflictFn: func(userID int64, conflictID int64, resolution string) (*models.Record, error) {
			require.Equal(t, models.ConflictResolutionLocal, resolution)
			return record, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.ResolveConflict(ctxWithUser(10), &pbv1.ResolveConflictRequest{
		ConflictId: 1,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
	require.Equal(t, int64(1), resp.Record.Id)
}

func TestResolveConflict_SuccessServer(t *testing.T) {
	record := sampleDomainRecord()
	mock := &mockSyncUseCase{
		resolveConflictFn: func(userID int64, conflictID int64, resolution string) (*models.Record, error) {
			require.Equal(t, models.ConflictResolutionServer, resolution)
			return record, nil
		},
	}
	svc := newTestSyncService(mock)

	resp, err := svc.ResolveConflict(ctxWithUser(10), &pbv1.ResolveConflictRequest{
		ConflictId: 1,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_SERVER,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Record)
}

func TestResolveConflict_NoAuth(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.ResolveConflict(context.Background(), &pbv1.ResolveConflictRequest{
		ConflictId: 1,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL,
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestResolveConflict_InvalidConflictID(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.ResolveConflict(ctxWithUser(1), &pbv1.ResolveConflictRequest{
		ConflictId: 0,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestResolveConflict_EmptyResolution(t *testing.T) {
	svc := newTestSyncService(&mockSyncUseCase{})
	_, err := svc.ResolveConflict(ctxWithUser(1), &pbv1.ResolveConflictRequest{
		ConflictId: 1,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_UNSPECIFIED,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestResolveConflict_UseCaseError(t *testing.T) {
	mock := &mockSyncUseCase{
		resolveConflictFn: func(userID int64, conflictID int64, resolution string) (*models.Record, error) {
			return nil, models.ErrConflictAlreadyResolved
		},
	}
	svc := newTestSyncService(mock)

	_, err := svc.ResolveConflict(ctxWithUser(1), &pbv1.ResolveConflictRequest{
		ConflictId: 1,
		Resolution: pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// --- protoRecordToDomain ---

func TestProtoRecordToDomain_Login(t *testing.T) {
	pb := &pbv1.Record{
		Id: 1, UserId: 10, Type: pbv1.RecordType_RECORD_TYPE_LOGIN,
		Name: "login", Metadata: "meta",
		Payload: &pbv1.Record_Login{
			Login: &pbv1.LoginPayload{Login: "user", Password: "pass"},
		},
		KeyVersion: 1, Revision: 2, DeviceId: "dev-1",
		CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}

	r := protoRecordToDomain(pb)
	require.Equal(t, models.RecordTypeLogin, r.Type)
	payload, ok := r.Payload.(models.LoginPayload)
	require.True(t, ok)
	require.Equal(t, "user", payload.Login)
	require.Equal(t, "pass", payload.Password)
}

func TestProtoRecordToDomain_Text(t *testing.T) {
	pb := &pbv1.Record{
		Id: 1, Type: pbv1.RecordType_RECORD_TYPE_TEXT,
		Payload: &pbv1.Record_Text{
			Text: &pbv1.TextPayload{Content: "hello"},
		},
		KeyVersion: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}

	r := protoRecordToDomain(pb)
	require.Equal(t, models.RecordTypeText, r.Type)
	payload, ok := r.Payload.(models.TextPayload)
	require.True(t, ok)
	require.Equal(t, "hello", payload.Content)
}

func TestProtoRecordToDomain_Binary(t *testing.T) {
	pb := &pbv1.Record{
		Id: 1, Type: pbv1.RecordType_RECORD_TYPE_BINARY,
		Payload: &pbv1.Record_Binary{
			Binary: &pbv1.BinaryPayload{},
		},
		KeyVersion: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}

	r := protoRecordToDomain(pb)
	require.Equal(t, models.RecordTypeBinary, r.Type)
	_, ok := r.Payload.(models.BinaryPayload)
	require.True(t, ok)
}

func TestProtoRecordToDomain_Card(t *testing.T) {
	pb := &pbv1.Record{
		Id: 1, Type: pbv1.RecordType_RECORD_TYPE_CARD,
		Payload: &pbv1.Record_Card{
			Card: &pbv1.CardPayload{Number: "4111", HolderName: "T", ExpiryDate: "12/25", Cvv: "123"},
		},
		KeyVersion: 1, CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}

	r := protoRecordToDomain(pb)
	require.Equal(t, models.RecordTypeCard, r.Type)
	payload, ok := r.Payload.(models.CardPayload)
	require.True(t, ok)
	require.Equal(t, "4111", payload.Number)
	require.Equal(t, "123", payload.CVV)
}

func TestProtoRecordToDomain_Nil(t *testing.T) {
	require.Nil(t, protoRecordToDomain(nil))
}

func TestProtoRecordToDomain_DeletedAt(t *testing.T) {
	now := time.Now()
	pb := &pbv1.Record{
		Id: 1, Type: pbv1.RecordType_RECORD_TYPE_TEXT,
		Payload: &pbv1.Record_Text{Text: &pbv1.TextPayload{Content: "x"}},
		KeyVersion: 1, DeletedAt: timestamppb.New(now),
		CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}

	r := protoRecordToDomain(pb)
	require.NotNil(t, r.DeletedAt)
}

// --- protoResolutionToDomain ---

func TestProtoResolutionToDomain(t *testing.T) {
	tests := []struct {
		name string
		in   pbv1.ConflictResolution
		want string
	}{
		{"local", pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL, models.ConflictResolutionLocal},
		{"server", pbv1.ConflictResolution_CONFLICT_RESOLUTION_SERVER, models.ConflictResolutionServer},
		{"unspecified", pbv1.ConflictResolution_CONFLICT_RESOLUTION_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protoResolutionToDomain(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

// --- domainResolutionToProto ---

func TestDomainResolutionToProto(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pbv1.ConflictResolution
	}{
		{"local", models.ConflictResolutionLocal, pbv1.ConflictResolution_CONFLICT_RESOLUTION_LOCAL},
		{"server", models.ConflictResolutionServer, pbv1.ConflictResolution_CONFLICT_RESOLUTION_SERVER},
		{"unknown", "unknown", pbv1.ConflictResolution_CONFLICT_RESOLUTION_UNSPECIFIED},
		{"empty", "", pbv1.ConflictResolution_CONFLICT_RESOLUTION_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainResolutionToProto(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

// --- mapSyncError ---

func TestMapSyncError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"not found", models.ErrRecordNotFound, codes.NotFound},
		{"revision conflict", models.ErrRevisionConflict, codes.Aborted},
		{"conflict already resolved", models.ErrConflictAlreadyResolved, codes.FailedPrecondition},
		{"invalid conflict resolution", models.ErrInvalidConflictResolution, codes.InvalidArgument},
		{"already deleted", models.ErrAlreadyDeleted, codes.FailedPrecondition},
		{"empty record name", models.ErrEmptyRecordName, codes.InvalidArgument},
		{"empty device id", models.ErrEmptyDeviceID, codes.InvalidArgument},
		{"invalid record type", models.ErrInvalidRecordType, codes.InvalidArgument},
		{"nil payload", models.ErrNilPayload, codes.InvalidArgument},
		{"invalid key version", models.ErrInvalidKeyVersion, codes.InvalidArgument},
		{"unknown error", errors.New("something unexpected"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSyncError(tt.err)
			require.Equal(t, tt.want, status.Code(got))
		})
	}
}
