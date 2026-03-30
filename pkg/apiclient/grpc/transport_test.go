package grpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	apiclient "github.com/hydra13/gophkeeper/pkg/apiclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// ---------------------------------------------------------------------------
// Mock implementations of gRPC client interfaces
// ---------------------------------------------------------------------------

// --- AuthServiceClient mock ---

type mockAuthServiceClient struct {
	registerFn func(ctx context.Context, in *pbv1.RegisterRequest, opts ...grpc.CallOption) (*pbv1.RegisterResponse, error)
	loginFn    func(ctx context.Context, in *pbv1.LoginRequest, opts ...grpc.CallOption) (*pbv1.LoginResponse, error)
	refreshFn  func(ctx context.Context, in *pbv1.RefreshRequest, opts ...grpc.CallOption) (*pbv1.RefreshResponse, error)
	logoutFn   func(ctx context.Context, in *pbv1.LogoutRequest, opts ...grpc.CallOption) (*pbv1.LogoutResponse, error)
}

func (m *mockAuthServiceClient) Register(ctx context.Context, in *pbv1.RegisterRequest, opts ...grpc.CallOption) (*pbv1.RegisterResponse, error) {
	return m.registerFn(ctx, in, opts...)
}

func (m *mockAuthServiceClient) Login(ctx context.Context, in *pbv1.LoginRequest, opts ...grpc.CallOption) (*pbv1.LoginResponse, error) {
	return m.loginFn(ctx, in, opts...)
}

func (m *mockAuthServiceClient) Refresh(ctx context.Context, in *pbv1.RefreshRequest, opts ...grpc.CallOption) (*pbv1.RefreshResponse, error) {
	return m.refreshFn(ctx, in, opts...)
}

func (m *mockAuthServiceClient) Logout(ctx context.Context, in *pbv1.LogoutRequest, opts ...grpc.CallOption) (*pbv1.LogoutResponse, error) {
	return m.logoutFn(ctx, in, opts...)
}

// --- DataServiceClient mock ---

type mockDataServiceClient struct {
	createRecordFn func(ctx context.Context, in *pbv1.CreateRecordRequest, opts ...grpc.CallOption) (*pbv1.CreateRecordResponse, error)
	getRecordFn    func(ctx context.Context, in *pbv1.GetRecordRequest, opts ...grpc.CallOption) (*pbv1.GetRecordResponse, error)
	listRecordsFn  func(ctx context.Context, in *pbv1.ListRecordsRequest, opts ...grpc.CallOption) (*pbv1.ListRecordsResponse, error)
	updateRecordFn func(ctx context.Context, in *pbv1.UpdateRecordRequest, opts ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error)
	deleteRecordFn func(ctx context.Context, in *pbv1.DeleteRecordRequest, opts ...grpc.CallOption) (*pbv1.DeleteRecordResponse, error)
}

func (m *mockDataServiceClient) CreateRecord(ctx context.Context, in *pbv1.CreateRecordRequest, opts ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
	return m.createRecordFn(ctx, in, opts...)
}

func (m *mockDataServiceClient) GetRecord(ctx context.Context, in *pbv1.GetRecordRequest, opts ...grpc.CallOption) (*pbv1.GetRecordResponse, error) {
	return m.getRecordFn(ctx, in, opts...)
}

func (m *mockDataServiceClient) ListRecords(ctx context.Context, in *pbv1.ListRecordsRequest, opts ...grpc.CallOption) (*pbv1.ListRecordsResponse, error) {
	return m.listRecordsFn(ctx, in, opts...)
}

func (m *mockDataServiceClient) UpdateRecord(ctx context.Context, in *pbv1.UpdateRecordRequest, opts ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
	return m.updateRecordFn(ctx, in, opts...)
}

func (m *mockDataServiceClient) DeleteRecord(ctx context.Context, in *pbv1.DeleteRecordRequest, opts ...grpc.CallOption) (*pbv1.DeleteRecordResponse, error) {
	return m.deleteRecordFn(ctx, in, opts...)
}

// --- SyncServiceClient mock ---

type mockSyncServiceClient struct {
	pullFn           func(ctx context.Context, in *pbv1.PullRequest, opts ...grpc.CallOption) (*pbv1.PullResponse, error)
	pushFn           func(ctx context.Context, in *pbv1.PushRequest, opts ...grpc.CallOption) (*pbv1.PushResponse, error)
	getConflictsFn   func(ctx context.Context, in *pbv1.GetConflictsRequest, opts ...grpc.CallOption) (*pbv1.GetConflictsResponse, error)
	resolveConflictFn func(ctx context.Context, in *pbv1.ResolveConflictRequest, opts ...grpc.CallOption) (*pbv1.ResolveConflictResponse, error)
}

func (m *mockSyncServiceClient) Pull(ctx context.Context, in *pbv1.PullRequest, opts ...grpc.CallOption) (*pbv1.PullResponse, error) {
	return m.pullFn(ctx, in, opts...)
}

func (m *mockSyncServiceClient) Push(ctx context.Context, in *pbv1.PushRequest, opts ...grpc.CallOption) (*pbv1.PushResponse, error) {
	return m.pushFn(ctx, in, opts...)
}

func (m *mockSyncServiceClient) GetConflicts(ctx context.Context, in *pbv1.GetConflictsRequest, opts ...grpc.CallOption) (*pbv1.GetConflictsResponse, error) {
	return m.getConflictsFn(ctx, in, opts...)
}

func (m *mockSyncServiceClient) ResolveConflict(ctx context.Context, in *pbv1.ResolveConflictRequest, opts ...grpc.CallOption) (*pbv1.ResolveConflictResponse, error) {
	return m.resolveConflictFn(ctx, in, opts...)
}

// --- UploadsServiceClient mock ---

type mockUploadsServiceClient struct {
	createUploadSessionFn  func(ctx context.Context, in *pbv1.CreateUploadSessionRequest, opts ...grpc.CallOption) (*pbv1.CreateUploadSessionResponse, error)
	uploadChunkFn          func(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error)
	getUploadStatusFn      func(ctx context.Context, in *pbv1.GetUploadStatusRequest, opts ...grpc.CallOption) (*pbv1.GetUploadStatusResponse, error)
	createDownloadSessionFn func(ctx context.Context, in *pbv1.CreateDownloadSessionRequest, opts ...grpc.CallOption) (*pbv1.CreateDownloadSessionResponse, error)
	downloadChunkFn        func(ctx context.Context, in *pbv1.DownloadChunkRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error)
	confirmChunkFn         func(ctx context.Context, in *pbv1.ConfirmChunkRequest, opts ...grpc.CallOption) (*pbv1.ConfirmChunkResponse, error)
	getDownloadStatusFn    func(ctx context.Context, in *pbv1.GetDownloadStatusRequest, opts ...grpc.CallOption) (*pbv1.GetDownloadStatusResponse, error)
}

func (m *mockUploadsServiceClient) CreateUploadSession(ctx context.Context, in *pbv1.CreateUploadSessionRequest, opts ...grpc.CallOption) (*pbv1.CreateUploadSessionResponse, error) {
	return m.createUploadSessionFn(ctx, in, opts...)
}

func (m *mockUploadsServiceClient) UploadChunk(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error) {
	return m.uploadChunkFn(ctx, opts...)
}

func (m *mockUploadsServiceClient) GetUploadStatus(ctx context.Context, in *pbv1.GetUploadStatusRequest, opts ...grpc.CallOption) (*pbv1.GetUploadStatusResponse, error) {
	return m.getUploadStatusFn(ctx, in, opts...)
}

func (m *mockUploadsServiceClient) CreateDownloadSession(ctx context.Context, in *pbv1.CreateDownloadSessionRequest, opts ...grpc.CallOption) (*pbv1.CreateDownloadSessionResponse, error) {
	return m.createDownloadSessionFn(ctx, in, opts...)
}

func (m *mockUploadsServiceClient) DownloadChunk(ctx context.Context, in *pbv1.DownloadChunkRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error) {
	return m.downloadChunkFn(ctx, in, opts...)
}

func (m *mockUploadsServiceClient) ConfirmChunk(ctx context.Context, in *pbv1.ConfirmChunkRequest, opts ...grpc.CallOption) (*pbv1.ConfirmChunkResponse, error) {
	return m.confirmChunkFn(ctx, in, opts...)
}

func (m *mockUploadsServiceClient) GetDownloadStatus(ctx context.Context, in *pbv1.GetDownloadStatusRequest, opts ...grpc.CallOption) (*pbv1.GetDownloadStatusResponse, error) {
	return m.getDownloadStatusFn(ctx, in, opts...)
}

// --- Stream mocks ---

type mockUploadStream struct {
	sendFn         func(req *pbv1.UploadChunkRequest) error
	closeAndRecvFn func() (*pbv1.UploadChunkResponse, error)
}

func (m *mockUploadStream) Send(req *pbv1.UploadChunkRequest) error {
	return m.sendFn(req)
}

func (m *mockUploadStream) CloseAndRecv() (*pbv1.UploadChunkResponse, error) {
	return m.closeAndRecvFn()
}

func (m *mockUploadStream) Header() (metadata.MD, error)         { return nil, nil }
func (m *mockUploadStream) Trailer() metadata.MD                  { return nil }
func (m *mockUploadStream) CloseSend() error                      { return nil }
func (m *mockUploadStream) Context() context.Context              { return context.Background() }
func (m *mockUploadStream) SendMsg(msg interface{}) error         { return nil }
func (m *mockUploadStream) RecvMsg(msg interface{}) error         { return nil }

type mockDownloadStream struct {
	recvFn func() (*pbv1.DownloadChunkResponse, error)
}

func (m *mockDownloadStream) Recv() (*pbv1.DownloadChunkResponse, error) {
	return m.recvFn()
}

func (m *mockDownloadStream) Header() (metadata.MD, error)         { return nil, nil }
func (m *mockDownloadStream) Trailer() metadata.MD                  { return nil }
func (m *mockDownloadStream) CloseSend() error                      { return nil }
func (m *mockDownloadStream) Context() context.Context              { return context.Background() }
func (m *mockDownloadStream) SendMsg(msg interface{}) error         { return nil }
func (m *mockDownloadStream) RecvMsg(msg interface{}) error         { return nil }

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type testClients struct {
	auth  *mockAuthServiceClient
	data  *mockDataServiceClient
	sync  *mockSyncServiceClient
	upload *mockUploadsServiceClient
}

func newTestClient(token string) (*Client, *testClients) {
	auth := &mockAuthServiceClient{}
	data := &mockDataServiceClient{}
	sync := &mockSyncServiceClient{}
	upload := &mockUploadsServiceClient{}

	c := &Client{
		authConn:     nil, // nil is ok for tests — Close handles it
		authClient:   auth,
		dataClient:   data,
		syncClient:   sync,
		uploadClient: upload,
		accessToken:  token,
	}

	return c, &testClients{auth: auth, data: data, sync: sync, upload: upload}
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.registerFn = func(_ context.Context, req *pbv1.RegisterRequest, _ ...grpc.CallOption) (*pbv1.RegisterResponse, error) {
		assert.Equal(t, "user@test.com", req.Email)
		assert.Equal(t, "secret", req.Password)
		return &pbv1.RegisterResponse{UserId: 42}, nil
	}

	id, err := c.Register(context.Background(), "user@test.com", "secret")
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestRegister_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.registerFn = func(_ context.Context, _ *pbv1.RegisterRequest, _ ...grpc.CallOption) (*pbv1.RegisterResponse, error) {
		return nil, fmt.Errorf("already exists")
	}

	_, err := c.Register(context.Background(), "dup@test.com", "pw")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc register")
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.loginFn = func(_ context.Context, req *pbv1.LoginRequest, _ ...grpc.CallOption) (*pbv1.LoginResponse, error) {
		assert.Equal(t, "a@b.c", req.Email)
		assert.Equal(t, "pass", req.Password)
		assert.Equal(t, "dev-1", req.DeviceId)
		return &pbv1.LoginResponse{AccessToken: "at", RefreshToken: "rt"}, nil
	}

	at, rt, err := c.Login(context.Background(), "a@b.c", "pass", "dev-1", "MyPhone", "cli")
	require.NoError(t, err)
	assert.Equal(t, "at", at)
	assert.Equal(t, "rt", rt)
	assert.Equal(t, "at", c.accessToken)
}

func TestLogin_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.loginFn = func(_ context.Context, _ *pbv1.LoginRequest, _ ...grpc.CallOption) (*pbv1.LoginResponse, error) {
		return nil, fmt.Errorf("invalid credentials")
	}

	_, _, err := c.Login(context.Background(), "x", "y", "d", "n", "c")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc login")
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("old-token")
	mocks.auth.refreshFn = func(_ context.Context, req *pbv1.RefreshRequest, _ ...grpc.CallOption) (*pbv1.RefreshResponse, error) {
		assert.Equal(t, "refresh-rt", req.RefreshToken)
		return &pbv1.RefreshResponse{AccessToken: "new-at", RefreshToken: "new-rt"}, nil
	}

	at, rt, err := c.Refresh(context.Background(), "refresh-rt")
	require.NoError(t, err)
	assert.Equal(t, "new-at", at)
	assert.Equal(t, "new-rt", rt)
	assert.Equal(t, "new-at", c.accessToken)
}

func TestRefresh_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.refreshFn = func(_ context.Context, _ *pbv1.RefreshRequest, _ ...grpc.CallOption) (*pbv1.RefreshResponse, error) {
		return nil, fmt.Errorf("token expired")
	}

	_, _, err := c.Refresh(context.Background(), "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc refresh")
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("my-token")
	mocks.auth.logoutFn = func(_ context.Context, _ *pbv1.LogoutRequest, _ ...grpc.CallOption) (*pbv1.LogoutResponse, error) {
		return &pbv1.LogoutResponse{}, nil
	}

	err := c.Logout(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "", c.accessToken, "accessToken should be cleared after logout")
}

func TestLogout_NoAuthContext(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("")
	mocks.auth.logoutFn = func(_ context.Context, _ *pbv1.LogoutRequest, _ ...grpc.CallOption) (*pbv1.LogoutResponse, error) {
		return &pbv1.LogoutResponse{}, nil
	}

	err := c.Logout(context.Background())
	require.NoError(t, err)
}

func TestLogout_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("tok")
	mocks.auth.logoutFn = func(_ context.Context, _ *pbv1.LogoutRequest, _ ...grpc.CallOption) (*pbv1.LogoutResponse, error) {
		return nil, fmt.Errorf("unauthorized")
	}

	err := c.Logout(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc logout")
	assert.Equal(t, "tok", c.accessToken, "accessToken should NOT be cleared on error")
}

// ---------------------------------------------------------------------------
// CreateRecord
// ---------------------------------------------------------------------------

func makeTestRecord(payload models.RecordPayload) *models.Record {
	return &models.Record{
		ID:             0,
		UserID:         1,
		Type:           payload.RecordType(),
		Name:           "test record",
		Metadata:       `{"env":"test"}`,
		DeviceID:       "dev-test",
		KeyVersion:     1,
		PayloadVersion: 1,
		Payload:        payload,
	}
}

func TestCreateRecord_Login(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.createRecordFn = func(_ context.Context, req *pbv1.CreateRecordRequest, _ ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
		assert.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, req.Type)
		assert.Equal(t, "test record", req.Name)
		assert.NotNil(t, req.Payload)
		return &pbv1.CreateRecordResponse{
			Record: &pbv1.Record{Id: 10, Name: "test record", Type: pbv1.RecordType_RECORD_TYPE_LOGIN,
				Payload: &pbv1.Record_Login{Login: &pbv1.LoginPayload{Login: "u", Password: "p"}}},
		}, nil
	}

	rec := makeTestRecord(models.LoginPayload{Login: "u", Password: "p"})
	result, err := c.CreateRecord(context.Background(), rec)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), result.ID)
}

func TestCreateRecord_Text(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.createRecordFn = func(_ context.Context, req *pbv1.CreateRecordRequest, _ ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
		assert.Equal(t, pbv1.RecordType_RECORD_TYPE_TEXT, req.Type)
		return &pbv1.CreateRecordResponse{
			Record: &pbv1.Record{Id: 11, Type: pbv1.RecordType_RECORD_TYPE_TEXT,
				Payload: &pbv1.Record_Text{Text: &pbv1.TextPayload{Content: "hello"}}},
		}, nil
	}

	rec := makeTestRecord(models.TextPayload{Content: "hello"})
	result, err := c.CreateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(11), result.ID)
}

func TestCreateRecord_Binary(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.createRecordFn = func(_ context.Context, req *pbv1.CreateRecordRequest, _ ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
		assert.Equal(t, pbv1.RecordType_RECORD_TYPE_BINARY, req.Type)
		return &pbv1.CreateRecordResponse{
			Record: &pbv1.Record{Id: 12, Type: pbv1.RecordType_RECORD_TYPE_BINARY,
				Payload: &pbv1.Record_Binary{Binary: &pbv1.BinaryPayload{}}},
		}, nil
	}

	rec := makeTestRecord(models.BinaryPayload{Data: []byte{1, 2}})
	result, err := c.CreateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(12), result.ID)
}

func TestCreateRecord_Card(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.createRecordFn = func(_ context.Context, req *pbv1.CreateRecordRequest, _ ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
		assert.Equal(t, pbv1.RecordType_RECORD_TYPE_CARD, req.Type)
		return &pbv1.CreateRecordResponse{
			Record: &pbv1.Record{Id: 13, Type: pbv1.RecordType_RECORD_TYPE_CARD,
				Payload: &pbv1.Record_Card{Card: &pbv1.CardPayload{Number: "4111"}}},
		}, nil
	}

	rec := makeTestRecord(models.CardPayload{Number: "4111", HolderName: "Test", ExpiryDate: "01/30", CVV: "000"})
	result, err := c.CreateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(13), result.ID)
}

func TestCreateRecord_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.createRecordFn = func(_ context.Context, _ *pbv1.CreateRecordRequest, _ ...grpc.CallOption) (*pbv1.CreateRecordResponse, error) {
		return nil, fmt.Errorf("internal error")
	}

	rec := makeTestRecord(models.LoginPayload{Login: "u", Password: "p"})
	_, err := c.CreateRecord(context.Background(), rec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc create record")
}

// ---------------------------------------------------------------------------
// GetRecord
// ---------------------------------------------------------------------------

func TestGetRecord_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.getRecordFn = func(_ context.Context, req *pbv1.GetRecordRequest, _ ...grpc.CallOption) (*pbv1.GetRecordResponse, error) {
		assert.Equal(t, int64(99), req.Id)
		return &pbv1.GetRecordResponse{
			Record: &pbv1.Record{Id: 99, Name: "found", Type: pbv1.RecordType_RECORD_TYPE_LOGIN},
		}, nil
	}

	result, err := c.GetRecord(context.Background(), 99)
	require.NoError(t, err)
	assert.Equal(t, int64(99), result.ID)
	assert.Equal(t, "found", result.Name)
}

func TestGetRecord_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.getRecordFn = func(_ context.Context, _ *pbv1.GetRecordRequest, _ ...grpc.CallOption) (*pbv1.GetRecordResponse, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := c.GetRecord(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc get record")
}

// ---------------------------------------------------------------------------
// ListRecords
// ---------------------------------------------------------------------------

func TestListRecords_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.listRecordsFn = func(_ context.Context, req *pbv1.ListRecordsRequest, _ ...grpc.CallOption) (*pbv1.ListRecordsResponse, error) {
		assert.Equal(t, pbv1.RecordType_RECORD_TYPE_LOGIN, req.Type)
		assert.True(t, req.IncludeDeleted)
		return &pbv1.ListRecordsResponse{
			Records: []*pbv1.Record{
				{Id: 1, Name: "r1", Type: pbv1.RecordType_RECORD_TYPE_LOGIN},
				{Id: 2, Name: "r2", Type: pbv1.RecordType_RECORD_TYPE_LOGIN},
			},
		}, nil
	}

	records, err := c.ListRecords(context.Background(), models.RecordTypeLogin, true)
	require.NoError(t, err)
	require.Len(t, records, 2)
	assert.Equal(t, int64(1), records[0].ID)
	assert.Equal(t, int64(2), records[1].ID)
}

func TestListRecords_Empty(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.listRecordsFn = func(_ context.Context, _ *pbv1.ListRecordsRequest, _ ...grpc.CallOption) (*pbv1.ListRecordsResponse, error) {
		return &pbv1.ListRecordsResponse{Records: []*pbv1.Record{}}, nil
	}

	records, err := c.ListRecords(context.Background(), models.RecordTypeText, false)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestListRecords_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.listRecordsFn = func(_ context.Context, _ *pbv1.ListRecordsRequest, _ ...grpc.CallOption) (*pbv1.ListRecordsResponse, error) {
		return nil, fmt.Errorf("denied")
	}

	_, err := c.ListRecords(context.Background(), models.RecordTypeCard, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc list records")
}

// ---------------------------------------------------------------------------
// UpdateRecord
// ---------------------------------------------------------------------------

func TestUpdateRecord_Login(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.updateRecordFn = func(_ context.Context, req *pbv1.UpdateRecordRequest, _ ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
		assert.Equal(t, int64(10), req.Id)
		assert.Equal(t, "updated", req.Name)
		return &pbv1.UpdateRecordResponse{
			Record: &pbv1.Record{Id: 10, Name: "updated", Revision: 2,
				Type: pbv1.RecordType_RECORD_TYPE_LOGIN,
				Payload: &pbv1.Record_Login{Login: &pbv1.LoginPayload{Login: "new", Password: "newp"}}},
		}, nil
	}

	rec := &models.Record{
		ID:       10,
		Name:     "updated",
		Type:     models.RecordTypeLogin,
		Revision: 1,
		Payload:  models.LoginPayload{Login: "new", Password: "newp"},
	}
	result, err := c.UpdateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(10), result.ID)
	assert.Equal(t, int64(2), result.Revision)
}

func TestUpdateRecord_Text(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.updateRecordFn = func(_ context.Context, _ *pbv1.UpdateRecordRequest, _ ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
		return &pbv1.UpdateRecordResponse{
			Record: &pbv1.Record{Id: 20, Type: pbv1.RecordType_RECORD_TYPE_TEXT,
				Payload: &pbv1.Record_Text{Text: &pbv1.TextPayload{Content: "new"}}},
		}, nil
	}

	rec := &models.Record{ID: 20, Type: models.RecordTypeText, Payload: models.TextPayload{Content: "new"}}
	result, err := c.UpdateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(20), result.ID)
}

func TestUpdateRecord_Card(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.updateRecordFn = func(_ context.Context, _ *pbv1.UpdateRecordRequest, _ ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
		return &pbv1.UpdateRecordResponse{
			Record: &pbv1.Record{Id: 30, Type: pbv1.RecordType_RECORD_TYPE_CARD,
				Payload: &pbv1.Record_Card{Card: &pbv1.CardPayload{Number: "9999"}}},
		}, nil
	}

	rec := &models.Record{ID: 30, Type: models.RecordTypeCard, Payload: models.CardPayload{Number: "9999"}}
	result, err := c.UpdateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(30), result.ID)
}

func TestUpdateRecord_Binary(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.updateRecordFn = func(_ context.Context, _ *pbv1.UpdateRecordRequest, _ ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
		return &pbv1.UpdateRecordResponse{
			Record: &pbv1.Record{Id: 40, Type: pbv1.RecordType_RECORD_TYPE_BINARY,
				Payload: &pbv1.Record_Binary{Binary: &pbv1.BinaryPayload{}}},
		}, nil
	}

	rec := &models.Record{ID: 40, Type: models.RecordTypeBinary, Payload: models.BinaryPayload{}}
	result, err := c.UpdateRecord(context.Background(), rec)
	require.NoError(t, err)
	assert.Equal(t, int64(40), result.ID)
}

func TestUpdateRecord_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.updateRecordFn = func(_ context.Context, _ *pbv1.UpdateRecordRequest, _ ...grpc.CallOption) (*pbv1.UpdateRecordResponse, error) {
		return nil, fmt.Errorf("conflict")
	}

	rec := &models.Record{ID: 1, Type: models.RecordTypeLogin, Payload: models.LoginPayload{}}
	_, err := c.UpdateRecord(context.Background(), rec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc update record")
}

// ---------------------------------------------------------------------------
// DeleteRecord
// ---------------------------------------------------------------------------

func TestDeleteRecord_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.deleteRecordFn = func(_ context.Context, req *pbv1.DeleteRecordRequest, _ ...grpc.CallOption) (*pbv1.DeleteRecordResponse, error) {
		assert.Equal(t, int64(77), req.Id)
		return &pbv1.DeleteRecordResponse{}, nil
	}

	err := c.DeleteRecord(context.Background(), 77)
	require.NoError(t, err)
}

func TestDeleteRecord_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.data.deleteRecordFn = func(_ context.Context, _ *pbv1.DeleteRecordRequest, _ ...grpc.CallOption) (*pbv1.DeleteRecordResponse, error) {
		return nil, fmt.Errorf("not found")
	}

	err := c.DeleteRecord(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc delete record")
}

// ---------------------------------------------------------------------------
// Pull
// ---------------------------------------------------------------------------

func TestPull_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pullFn = func(_ context.Context, req *pbv1.PullRequest, _ ...grpc.CallOption) (*pbv1.PullResponse, error) {
		assert.Equal(t, int64(5), req.SinceRevision)
		assert.Equal(t, "dev-1", req.DeviceId)
		assert.Equal(t, int32(50), req.Limit)
		return &pbv1.PullResponse{
			HasMore:      true,
			NextRevision: 20,
			Records: []*pbv1.Record{
				{Id: 10, Name: "r1", Type: pbv1.RecordType_RECORD_TYPE_TEXT},
			},
			Conflicts: []*pbv1.SyncConflict{
				{Id: 1, RecordId: 10, LocalRevision: 3, ServerRevision: 5},
			},
		}, nil
	}

	result, err := c.Pull(context.Background(), 5, "dev-1", 50)
	require.NoError(t, err)
	assert.True(t, result.HasMore)
	assert.Equal(t, int64(20), result.NextRevision)
	require.Len(t, result.Records, 1)
	assert.Equal(t, int64(10), result.Records[0].ID)
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, int64(10), result.Conflicts[0].RecordID)
}

func TestPull_Empty(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pullFn = func(_ context.Context, _ *pbv1.PullRequest, _ ...grpc.CallOption) (*pbv1.PullResponse, error) {
		return &pbv1.PullResponse{HasMore: false, NextRevision: 0}, nil
	}

	result, err := c.Pull(context.Background(), 0, "", 100)
	require.NoError(t, err)
	assert.False(t, result.HasMore)
	assert.Empty(t, result.Records)
	assert.Empty(t, result.Conflicts)
}

func TestPull_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pullFn = func(_ context.Context, _ *pbv1.PullRequest, _ ...grpc.CallOption) (*pbv1.PullResponse, error) {
		return nil, fmt.Errorf("server error")
	}

	_, err := c.Pull(context.Background(), 0, "", 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc pull")
}

// ---------------------------------------------------------------------------
// Push
// ---------------------------------------------------------------------------

func TestPush_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pushFn = func(_ context.Context, req *pbv1.PushRequest, _ ...grpc.CallOption) (*pbv1.PushResponse, error) {
		assert.Equal(t, "dev-2", req.DeviceId)
		require.Len(t, req.Changes, 1)
		return &pbv1.PushResponse{
			Accepted: []*pbv1.RecordRevision{
				{RecordId: 10, Revision: 11, DeviceId: "dev-2"},
			},
			Conflicts: []*pbv1.SyncConflict{
				{Id: 5, RecordId: 20, LocalRevision: 3, ServerRevision: 7},
			},
		}, nil
	}

	changes := []apiclient.PendingChange{
		{
			Record:       &models.Record{ID: 10, Name: "push", Type: models.RecordTypeLogin},
			Deleted:      false,
			BaseRevision: 10,
		},
	}

	result, err := c.Push(context.Background(), changes, "dev-2")
	require.NoError(t, err)
	require.Len(t, result.Accepted, 1)
	assert.Equal(t, int64(10), result.Accepted[0].RecordID)
	assert.Equal(t, int64(11), result.Accepted[0].Revision)
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, int64(20), result.Conflicts[0].RecordID)
}

func TestPush_Empty(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pushFn = func(_ context.Context, _ *pbv1.PushRequest, _ ...grpc.CallOption) (*pbv1.PushResponse, error) {
		return &pbv1.PushResponse{}, nil
	}

	result, err := c.Push(context.Background(), nil, "dev")
	require.NoError(t, err)
	assert.Empty(t, result.Accepted)
	assert.Empty(t, result.Conflicts)
}

func TestPush_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.sync.pushFn = func(_ context.Context, _ *pbv1.PushRequest, _ ...grpc.CallOption) (*pbv1.PushResponse, error) {
		return nil, fmt.Errorf("rejected")
	}

	_, err := c.Push(context.Background(), []apiclient.PendingChange{{}}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc push")
}

// ---------------------------------------------------------------------------
// CreateUploadSession
// ---------------------------------------------------------------------------

func TestCreateUploadSession_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.createUploadSessionFn = func(_ context.Context, req *pbv1.CreateUploadSessionRequest, _ ...grpc.CallOption) (*pbv1.CreateUploadSessionResponse, error) {
		assert.Equal(t, int64(5), req.RecordId)
		assert.Equal(t, int64(10), req.TotalChunks)
		assert.Equal(t, int64(4096), req.ChunkSize)
		assert.Equal(t, int64(40960), req.TotalSize)
		assert.Equal(t, int64(2), req.KeyVersion)
		return &pbv1.CreateUploadSessionResponse{UploadId: 100}, nil
	}

	id, err := c.CreateUploadSession(context.Background(), 5, 10, 4096, 40960, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(100), id)
}

func TestCreateUploadSession_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.createUploadSessionFn = func(_ context.Context, _ *pbv1.CreateUploadSessionRequest, _ ...grpc.CallOption) (*pbv1.CreateUploadSessionResponse, error) {
		return nil, fmt.Errorf("quota exceeded")
	}

	_, err := c.CreateUploadSession(context.Background(), 1, 1, 1, 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc create upload session")
}

// ---------------------------------------------------------------------------
// UploadChunk
// ---------------------------------------------------------------------------

func TestUploadChunk_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.uploadChunkFn = func(_ context.Context, _ ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error) {
		return &mockUploadStream{
			sendFn: func(req *pbv1.UploadChunkRequest) error {
				assert.Equal(t, int64(100), req.UploadId)
				assert.Equal(t, int64(0), req.ChunkIndex)
				assert.Equal(t, []byte("data"), req.Data)
				return nil
			},
			closeAndRecvFn: func() (*pbv1.UploadChunkResponse, error) {
				return &pbv1.UploadChunkResponse{}, nil
			},
		}, nil
	}

	err := c.UploadChunk(context.Background(), 100, 0, []byte("data"))
	require.NoError(t, err)
}

func TestUploadChunk_StreamError(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.uploadChunkFn = func(_ context.Context, _ ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error) {
		return nil, fmt.Errorf("stream unavailable")
	}

	err := c.UploadChunk(context.Background(), 1, 0, []byte("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc upload chunk stream")
}

func TestUploadChunk_SendError(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.uploadChunkFn = func(_ context.Context, _ ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error) {
		return &mockUploadStream{
			sendFn: func(_ *pbv1.UploadChunkRequest) error {
				return fmt.Errorf("network failure")
			},
		}, nil
	}

	err := c.UploadChunk(context.Background(), 1, 0, []byte("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc upload chunk send")
}

func TestUploadChunk_CloseError(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.uploadChunkFn = func(_ context.Context, _ ...grpc.CallOption) (grpc.ClientStreamingClient[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse], error) {
		return &mockUploadStream{
			sendFn: func(_ *pbv1.UploadChunkRequest) error { return nil },
			closeAndRecvFn: func() (*pbv1.UploadChunkResponse, error) {
				return nil, fmt.Errorf("server closed")
			},
		}, nil
	}

	err := c.UploadChunk(context.Background(), 1, 0, []byte("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc upload chunk close")
}

// ---------------------------------------------------------------------------
// GetUploadStatus
// ---------------------------------------------------------------------------

func TestGetUploadStatus_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.getUploadStatusFn = func(_ context.Context, req *pbv1.GetUploadStatusRequest, _ ...grpc.CallOption) (*pbv1.GetUploadStatusResponse, error) {
		assert.Equal(t, int64(42), req.UploadId)
		return &pbv1.GetUploadStatusResponse{
			UploadId:       42,
			Status:         pbv1.UploadStatus_UPLOAD_STATUS_PENDING,
			TotalChunks:    10,
			ReceivedChunks: 5,
			MissingChunks:  []int64{5, 6, 7, 8, 9},
		}, nil
	}

	status, err := c.GetUploadStatus(context.Background(), 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), status.UploadID)
	assert.Equal(t, "UPLOAD_STATUS_PENDING", status.Status)
	assert.Equal(t, int64(10), status.TotalChunks)
	assert.Equal(t, int64(5), status.ReceivedChunks)
}

func TestGetUploadStatus_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.getUploadStatusFn = func(_ context.Context, _ *pbv1.GetUploadStatusRequest, _ ...grpc.CallOption) (*pbv1.GetUploadStatusResponse, error) {
		return nil, fmt.Errorf("not found")
	}

	_, err := c.GetUploadStatus(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc get upload status")
}

// ---------------------------------------------------------------------------
// CreateDownloadSession
// ---------------------------------------------------------------------------

func TestCreateDownloadSession_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.createDownloadSessionFn = func(_ context.Context, req *pbv1.CreateDownloadSessionRequest, _ ...grpc.CallOption) (*pbv1.CreateDownloadSessionResponse, error) {
		assert.Equal(t, int64(7), req.RecordId)
		return &pbv1.CreateDownloadSessionResponse{DownloadId: 50, TotalChunks: 3}, nil
	}

	downloadID, totalChunks, err := c.CreateDownloadSession(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(50), downloadID)
	assert.Equal(t, int64(3), totalChunks)
}

func TestCreateDownloadSession_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.createDownloadSessionFn = func(_ context.Context, _ *pbv1.CreateDownloadSessionRequest, _ ...grpc.CallOption) (*pbv1.CreateDownloadSessionResponse, error) {
		return nil, fmt.Errorf("unavailable")
	}

	_, _, err := c.CreateDownloadSession(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc create download session")
}

// ---------------------------------------------------------------------------
// DownloadChunk
// ---------------------------------------------------------------------------

func TestDownloadChunk_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.downloadChunkFn = func(_ context.Context, req *pbv1.DownloadChunkRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error) {
		assert.Equal(t, int64(50), req.DownloadId)
		assert.Equal(t, int64(2), req.ChunkIndex)
		return &mockDownloadStream{
			recvFn: func() (*pbv1.DownloadChunkResponse, error) {
				return &pbv1.DownloadChunkResponse{Data: []byte("chunk-data")}, nil
			},
		}, nil
	}

	data, err := c.DownloadChunk(context.Background(), 50, 2)
	require.NoError(t, err)
	assert.Equal(t, []byte("chunk-data"), data)
}

func TestDownloadChunk_EOF(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.downloadChunkFn = func(_ context.Context, _ *pbv1.DownloadChunkRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error) {
		return &mockDownloadStream{
			recvFn: func() (*pbv1.DownloadChunkResponse, error) {
				return nil, io.EOF
			},
		}, nil
	}

	data, err := c.DownloadChunk(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Nil(t, data, "EOF should return nil data without error")
}

func TestDownloadChunk_StreamError(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.downloadChunkFn = func(_ context.Context, _ *pbv1.DownloadChunkRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error) {
		return nil, fmt.Errorf("stream error")
	}

	_, err := c.DownloadChunk(context.Background(), 1, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc download chunk")
}

func TestDownloadChunk_RecvError(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.downloadChunkFn = func(_ context.Context, _ *pbv1.DownloadChunkRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[pbv1.DownloadChunkResponse], error) {
		return &mockDownloadStream{
			recvFn: func() (*pbv1.DownloadChunkResponse, error) {
				return nil, fmt.Errorf("corrupted")
			},
		}, nil
	}

	_, err := c.DownloadChunk(context.Background(), 1, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc download chunk recv")
}

// ---------------------------------------------------------------------------
// ConfirmChunk
// ---------------------------------------------------------------------------

func TestConfirmChunk_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.confirmChunkFn = func(_ context.Context, req *pbv1.ConfirmChunkRequest, _ ...grpc.CallOption) (*pbv1.ConfirmChunkResponse, error) {
		assert.Equal(t, int64(50), req.DownloadId)
		assert.Equal(t, int64(1), req.ChunkIndex)
		return &pbv1.ConfirmChunkResponse{}, nil
	}

	err := c.ConfirmChunk(context.Background(), 50, 1)
	require.NoError(t, err)
}

func TestConfirmChunk_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.confirmChunkFn = func(_ context.Context, _ *pbv1.ConfirmChunkRequest, _ ...grpc.CallOption) (*pbv1.ConfirmChunkResponse, error) {
		return nil, fmt.Errorf("invalid chunk")
	}

	err := c.ConfirmChunk(context.Background(), 1, 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc confirm chunk")
}

// ---------------------------------------------------------------------------
// GetDownloadStatus
// ---------------------------------------------------------------------------

func TestGetDownloadStatus_Success(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.getDownloadStatusFn = func(_ context.Context, req *pbv1.GetDownloadStatusRequest, _ ...grpc.CallOption) (*pbv1.GetDownloadStatusResponse, error) {
		assert.Equal(t, int64(50), req.DownloadId)
		return &pbv1.GetDownloadStatusResponse{
			DownloadId:      50,
			Status:          pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE,
			TotalChunks:     5,
			ConfirmedChunks: 3,
			RemainingChunks: []int64{3, 4},
		}, nil
	}

	status, err := c.GetDownloadStatus(context.Background(), 50)
	require.NoError(t, err)
	assert.Equal(t, int64(50), status.DownloadID)
	assert.Equal(t, "DOWNLOAD_STATUS_ACTIVE", status.Status)
	assert.Equal(t, int64(5), status.TotalChunks)
	assert.Equal(t, int64(3), status.ConfirmedChunks)
}

func TestGetDownloadStatus_Error(t *testing.T) {
	t.Parallel()
	c, mocks := newTestClient("token")
	mocks.upload.getDownloadStatusFn = func(_ context.Context, _ *pbv1.GetDownloadStatusRequest, _ ...grpc.CallOption) (*pbv1.GetDownloadStatusResponse, error) {
		return nil, fmt.Errorf("expired")
	}

	_, err := c.GetDownloadStatus(context.Background(), 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc get download status")
}

// ---------------------------------------------------------------------------
// SetAccessToken
// ---------------------------------------------------------------------------

func TestSetAccessToken(t *testing.T) {
	t.Parallel()
	c, _ := newTestClient("old")
	assert.Equal(t, "old", c.accessToken)
	c.SetAccessToken("new")
	assert.Equal(t, "new", c.accessToken)
}

// ---------------------------------------------------------------------------
// authContext
// ---------------------------------------------------------------------------

func TestAuthContext_WithToken(t *testing.T) {
	t.Parallel()
	c, _ := newTestClient("my-jwt")
	ctx := c.authContext(context.Background())

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok, "metadata should be present")
	values := md.Get("authorization")
	require.Len(t, values, 1)
	assert.Equal(t, "Bearer my-jwt", values[0])
}

func TestAuthContext_WithoutToken(t *testing.T) {
	t.Parallel()
	c, _ := newTestClient("")
	ctx := c.authContext(context.Background())

	_, ok := metadata.FromOutgoingContext(ctx)
	assert.False(t, ok, "no metadata should be added when token is empty")
}

// ---------------------------------------------------------------------------
// Close
// ---------------------------------------------------------------------------

func TestClose_NilConn(t *testing.T) {
	t.Parallel()
	c, _ := newTestClient("")
	err := c.Close()
	require.NoError(t, err, "Close with nil conn should return nil")
}

// ---------------------------------------------------------------------------
// TLS smoke tests: real TLS handshake with dev certificates
// ---------------------------------------------------------------------------

func TestTLSHandshake_WithDevCert(t *testing.T) {
	// Verify TLS handshake succeeds using the documented dev cert path.
	certPath := filepath.Join("..", "..", "..", "configs", "certs", "dev.crt")
	keyPath := filepath.Join("..", "..", "..", "configs", "certs", "dev.key")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Skip("configs/certs/dev.crt not found, skipping TLS smoke test")
	}

	// Start a TLS gRPC server on a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	creds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
	require.NoError(t, err)

	srv := grpc.NewServer(grpc.Creds(creds))
	go srv.Serve(lis)
	defer srv.Stop()

	// Connect client using the documented CA cert path.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := NewClient(ctx, Config{
		Address:     lis.Addr().String(),
		TLSCertFile: certPath,
	})
	require.NoError(t, err)
	defer client.Close()
}

func TestTLSHandshake_WrongCertFails(t *testing.T) {
	// Verify TLS handshake fails when client uses a wrong CA cert.
	certPath := filepath.Join("..", "..", "..", "configs", "certs", "dev.crt")
	keyPath := filepath.Join("..", "..", "..", "configs", "certs", "dev.key")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Skip("configs/certs/dev.crt not found, skipping TLS smoke test")
	}

	// Start TLS server with dev cert.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	creds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
	require.NoError(t, err)

	srv := grpc.NewServer(grpc.Creds(creds))
	go srv.Serve(lis)
	defer srv.Stop()

	// Generate a different self-signed cert to use as CA (wrong trust bundle).
	wrongCertPEM, _, err := generateTestCert()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	wrongCertPath := filepath.Join(tmpDir, "wrong-ca.crt")
	err = os.WriteFile(wrongCertPath, wrongCertPEM, 0644)
	require.NoError(t, err)

	// Client with wrong CA should fail during Register (TLS handshake error).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := NewClient(ctx, Config{
		Address:     lis.Addr().String(),
		TLSCertFile: wrongCertPath,
	})
	require.NoError(t, err)
	defer client.Close()

	// The TLS handshake error surfaces when making an actual RPC call.
	_, err = client.Register(ctx, "test@example.com", "password")
	require.Error(t, err, "expected TLS handshake error with wrong CA cert")
}
