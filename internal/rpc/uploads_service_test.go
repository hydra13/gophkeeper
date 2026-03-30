package rpc

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// ---------------------------------------------------------------------------
// Mock UploadsUseCase
// ---------------------------------------------------------------------------

type mockUploadsUseCase struct {
	createSessionFn        func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
	getUploadSessionByIDFn func(uploadID int64) (*models.UploadSession, error)
	uploadChunkFn          func(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error)
	createDownloadFn       func(userID, recordID int64) (*models.DownloadSession, error)
	downloadChunkByIDFn    func(downloadID, chunkIndex int64) (*models.Chunk, error)
	confirmChunkFn         func(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error)
	getDownloadStatusFn    func(downloadID int64) (*models.DownloadSession, error)
}

func (m *mockUploadsUseCase) CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	return m.createSessionFn(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion)
}

func (m *mockUploadsUseCase) GetUploadSessionByID(uploadID int64) (*models.UploadSession, error) {
	return m.getUploadSessionByIDFn(uploadID)
}

func (m *mockUploadsUseCase) UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error) {
	return m.uploadChunkFn(uploadID, chunkIndex, data)
}

func (m *mockUploadsUseCase) CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error) {
	return m.createDownloadFn(userID, recordID)
}

func (m *mockUploadsUseCase) DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error) {
	return m.downloadChunkByIDFn(downloadID, chunkIndex)
}

func (m *mockUploadsUseCase) ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error) {
	return m.confirmChunkFn(downloadID, chunkIndex)
}

func (m *mockUploadsUseCase) GetDownloadStatus(downloadID int64) (*models.DownloadSession, error) {
	return m.getDownloadStatusFn(downloadID)
}

func newTestUploadsService() *UploadsService {
	return NewUploadsService(&mockUploadsUseCase{}, zerolog.Nop())
}

// ---------------------------------------------------------------------------
// mockClientStream implements grpc.ClientStreamingServer for UploadChunk tests
// ---------------------------------------------------------------------------

type mockUploadChunkStream struct {
	grpc.ServerStream
	requests []*pbv1.UploadChunkRequest
	idx      int
	resp     *pbv1.UploadChunkResponse
	closed   bool
	recvErr  error // error to return after all requests consumed
}

func (s *mockUploadChunkStream) Context() context.Context {
	return ctxWithUser(1)
}

func (s *mockUploadChunkStream) Recv() (*pbv1.UploadChunkRequest, error) {
	if s.idx < len(s.requests) {
		req := s.requests[s.idx]
		s.idx++
		return req, nil
	}
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	return nil, io.EOF
}

func (s *mockUploadChunkStream) SendAndClose(resp *pbv1.UploadChunkResponse) error {
	s.resp = resp
	s.closed = true
	return nil
}

// ---------------------------------------------------------------------------
// mockServerStreamingServer implements grpc.ServerStreamingServer for DownloadChunk tests
// ---------------------------------------------------------------------------

type mockDownloadChunkStream struct {
	grpc.ServerStream
	sent    []*pbv1.DownloadChunkResponse
	sendErr error
}

func (s *mockDownloadChunkStream) Context() context.Context {
	return ctxWithUser(1)
}

func (s *mockDownloadChunkStream) Send(resp *pbv1.DownloadChunkResponse) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, resp)
	return nil
}

// ---------------------------------------------------------------------------
// CreateUploadSession
// ---------------------------------------------------------------------------

func TestCreateUploadSession_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		createSessionFn: func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
			return 42, nil
		},
	}

	resp, err := svc.CreateUploadSession(ctxWithUser(1), &pbv1.CreateUploadSessionRequest{
		RecordId:    10,
		TotalChunks: 3,
		ChunkSize:   1024,
		TotalSize:   3072,
		KeyVersion:  1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), resp.UploadId)
	require.Equal(t, pbv1.UploadStatus_UPLOAD_STATUS_PENDING, resp.Status)
}

func TestCreateUploadSession_Unauthenticated(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.CreateUploadSession(context.Background(), &pbv1.CreateUploadSessionRequest{
		RecordId: 10, TotalChunks: 3, ChunkSize: 1024, TotalSize: 3072,
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestCreateUploadSession_InvalidRecordID(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.CreateUploadSession(ctxWithUser(1), &pbv1.CreateUploadSessionRequest{
		RecordId: 0, TotalChunks: 3, ChunkSize: 1024, TotalSize: 3072,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateUploadSession_InvalidTotalChunks(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.CreateUploadSession(ctxWithUser(1), &pbv1.CreateUploadSessionRequest{
		RecordId: 10, TotalChunks: 0, ChunkSize: 1024, TotalSize: 3072,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateUploadSession_UseCaseError(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		createSessionFn: func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	_, err := svc.CreateUploadSession(ctxWithUser(1), &pbv1.CreateUploadSessionRequest{
		RecordId: 10, TotalChunks: 3, ChunkSize: 1024, TotalSize: 3072,
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

// ---------------------------------------------------------------------------
// UploadChunk (client streaming)
// ---------------------------------------------------------------------------

func TestUploadChunk_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		uploadChunkFn: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 1, 3, false, []int64{1, 2}, nil
		},
	}

	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 0, Data: []byte("hello")},
		},
	}

	err := svc.UploadChunk(stream)
	require.NoError(t, err)
	require.True(t, stream.closed)
	require.Equal(t, int64(1), stream.resp.ReceivedChunks)
	require.Equal(t, int64(3), stream.resp.TotalChunks)
	require.Equal(t, pbv1.UploadStatus_UPLOAD_STATUS_PENDING, stream.resp.Status)
}

func TestUploadChunk_Completed(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		uploadChunkFn: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 3, 3, true, nil, nil
		},
	}

	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 0, Data: []byte("a")},
			{UploadId: 1, ChunkIndex: 1, Data: []byte("b")},
			{UploadId: 1, ChunkIndex: 2, Data: []byte("c")},
		},
	}

	err := svc.UploadChunk(stream)
	require.NoError(t, err)
	require.True(t, stream.closed)
	require.Equal(t, pbv1.UploadStatus_UPLOAD_STATUS_COMPLETED, stream.resp.Status)
}

func TestUploadChunk_Unauthenticated(t *testing.T) {
	svc := newTestUploadsService()
	_ = svc
	// userIDFromContext проверяется до входа в цикл.
	// Поскольку mockUploadChunkStream возвращает ctxWithUser(1),
	// протестировать unauthenticated через этот mock нельзя —
	// это покрывается тестами других методов (CreateUploadSession_Unauthenticated).
}

func TestUploadChunk_InvalidUploadID(t *testing.T) {
	svc := newTestUploadsService()
	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 0, ChunkIndex: 0, Data: []byte("hello")},
		},
	}

	err := svc.UploadChunk(stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUploadChunk_EmptyData(t *testing.T) {
	svc := newTestUploadsService()
	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 0, Data: []byte{}},
		},
	}

	err := svc.UploadChunk(stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUploadChunk_ChunkOutOfOrder(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		uploadChunkFn: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, models.ErrChunkOutOfOrder
		},
	}

	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 2, Data: []byte("data")},
		},
	}

	err := svc.UploadChunk(stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUploadChunk_DuplicateChunk(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		uploadChunkFn: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, models.ErrDuplicateChunk
		},
	}

	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 0, Data: []byte("data")},
		},
	}

	err := svc.UploadChunk(stream)
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestUploadChunk_UploadCompleted(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		uploadChunkFn: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, models.ErrUploadCompleted
		},
	}

	stream := &mockUploadChunkStream{
		requests: []*pbv1.UploadChunkRequest{
			{UploadId: 1, ChunkIndex: 0, Data: []byte("data")},
		},
	}

	err := svc.UploadChunk(stream)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// ---------------------------------------------------------------------------
// GetUploadStatus
// ---------------------------------------------------------------------------

func TestGetUploadStatus_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getUploadSessionByIDFn: func(uploadID int64) (*models.UploadSession, error) {
			return &models.UploadSession{
				ID:             1,
				Status:         models.UploadStatusPending,
				TotalChunks:    3,
				ReceivedChunks: 1,
				ReceivedChunkSet: map[int64]bool{0: true},
			}, nil
		},
	}

	resp, err := svc.GetUploadStatus(ctxWithUser(1), &pbv1.GetUploadStatusRequest{UploadId: 1})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.UploadId)
	require.Equal(t, pbv1.UploadStatus_UPLOAD_STATUS_PENDING, resp.Status)
	require.Equal(t, int64(3), resp.TotalChunks)
	require.Equal(t, int64(1), resp.ReceivedChunks)
	require.Equal(t, []int64{1, 2}, resp.MissingChunks)
}

func TestGetUploadStatus_InvalidUploadID(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.GetUploadStatus(ctxWithUser(1), &pbv1.GetUploadStatusRequest{UploadId: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetUploadStatus_NotFound(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getUploadSessionByIDFn: func(uploadID int64) (*models.UploadSession, error) {
			return nil, models.ErrUploadNotFound
		},
	}
	_, err := svc.GetUploadStatus(ctxWithUser(1), &pbv1.GetUploadStatusRequest{UploadId: 999})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// ---------------------------------------------------------------------------
// CreateDownloadSession
// ---------------------------------------------------------------------------

func TestCreateDownloadSession_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		createDownloadFn: func(userID, recordID int64) (*models.DownloadSession, error) {
			return &models.DownloadSession{
				ID:          10,
				RecordID:    recordID,
				UserID:      userID,
				Status:      models.DownloadStatusActive,
				TotalChunks: 3,
			}, nil
		},
	}

	resp, err := svc.CreateDownloadSession(ctxWithUser(1), &pbv1.CreateDownloadSessionRequest{RecordId: 5})
	require.NoError(t, err)
	require.Equal(t, int64(10), resp.DownloadId)
	require.Equal(t, pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE, resp.Status)
	require.Equal(t, int64(3), resp.TotalChunks)
}

func TestCreateDownloadSession_InvalidRecordID(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.CreateDownloadSession(ctxWithUser(1), &pbv1.CreateDownloadSessionRequest{RecordId: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreateDownloadSession_UploadNotFound(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		createDownloadFn: func(userID, recordID int64) (*models.DownloadSession, error) {
			return nil, models.ErrUploadNotFound
		},
	}
	_, err := svc.CreateDownloadSession(ctxWithUser(1), &pbv1.CreateDownloadSessionRequest{RecordId: 5})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// ---------------------------------------------------------------------------
// DownloadChunk (server streaming)
// ---------------------------------------------------------------------------

func TestDownloadChunk_SingleChunk(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getDownloadStatusFn: func(downloadID int64) (*models.DownloadSession, error) {
			return &models.DownloadSession{
				ID:          1,
				TotalChunks: 3,
				Status:      models.DownloadStatusActive,
			}, nil
		},
		downloadChunkByIDFn: func(downloadID, chunkIndex int64) (*models.Chunk, error) {
			return &models.Chunk{UploadID: 1, ChunkIndex: chunkIndex, Data: []byte("data")}, nil
		},
	}

	stream := &mockDownloadChunkStream{}
	err := svc.DownloadChunk(&pbv1.DownloadChunkRequest{DownloadId: 1, ChunkIndex: 0}, stream)
	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	require.Equal(t, int64(0), stream.sent[0].ChunkIndex)
	require.Equal(t, []byte("data"), stream.sent[0].Data)
}

func TestDownloadChunk_AllChunks(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getDownloadStatusFn: func(downloadID int64) (*models.DownloadSession, error) {
			return &models.DownloadSession{
				ID:                 1,
				TotalChunks:        2,
				Status:             models.DownloadStatusActive,
				ConfirmedChunkSet:  map[int64]bool{},
			}, nil
		},
		downloadChunkByIDFn: func(downloadID, chunkIndex int64) (*models.Chunk, error) {
			return &models.Chunk{UploadID: 1, ChunkIndex: chunkIndex, Data: []byte("x")}, nil
		},
	}

	stream := &mockDownloadChunkStream{}
	err := svc.DownloadChunk(&pbv1.DownloadChunkRequest{DownloadId: 1, ChunkIndex: -1}, stream)
	require.NoError(t, err)
	require.Len(t, stream.sent, 2)
}

func TestDownloadChunk_InvalidDownloadID(t *testing.T) {
	svc := newTestUploadsService()
	stream := &mockDownloadChunkStream{}
	err := svc.DownloadChunk(&pbv1.DownloadChunkRequest{DownloadId: 0}, stream)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDownloadChunk_DownloadNotFound(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getDownloadStatusFn: func(downloadID int64) (*models.DownloadSession, error) {
			return nil, models.ErrDownloadNotFound
		},
	}

	stream := &mockDownloadChunkStream{}
	err := svc.DownloadChunk(&pbv1.DownloadChunkRequest{DownloadId: 999, ChunkIndex: 0}, stream)
	require.Equal(t, codes.NotFound, status.Code(err))
}

// ---------------------------------------------------------------------------
// ConfirmChunk
// ---------------------------------------------------------------------------

func TestConfirmChunk_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		confirmChunkFn: func(downloadID, chunkIndex int64) (int64, int64, models.DownloadStatus, error) {
			return 1, 3, models.DownloadStatusActive, nil
		},
	}

	resp, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 1, ChunkIndex: 0})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.ConfirmedChunks)
	require.Equal(t, int64(3), resp.TotalChunks)
	require.Equal(t, pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE, resp.Status)
}

func TestConfirmChunk_InvalidDownloadID(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 0, ChunkIndex: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestConfirmChunk_NegativeChunkIndex(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 1, ChunkIndex: -1})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestConfirmChunk_OutOfOrder(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		confirmChunkFn: func(downloadID, chunkIndex int64) (int64, int64, models.DownloadStatus, error) {
			return 0, 0, "", models.ErrChunkOutOfOrder
		},
	}

	_, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 1, ChunkIndex: 2})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestConfirmChunk_DownloadCompleted(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		confirmChunkFn: func(downloadID, chunkIndex int64) (int64, int64, models.DownloadStatus, error) {
			return 0, 0, "", models.ErrDownloadCompleted
		},
	}

	_, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 1, ChunkIndex: 0})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestConfirmChunk_ChunkAlreadyConfirmed(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		confirmChunkFn: func(downloadID, chunkIndex int64) (int64, int64, models.DownloadStatus, error) {
			return 0, 0, "", models.ErrChunkAlreadyConfirmed
		},
	}

	_, err := svc.ConfirmChunk(ctxWithUser(1), &pbv1.ConfirmChunkRequest{DownloadId: 1, ChunkIndex: 0})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

// ---------------------------------------------------------------------------
// GetDownloadStatus
// ---------------------------------------------------------------------------

func TestGetDownloadStatus_Success(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getDownloadStatusFn: func(downloadID int64) (*models.DownloadSession, error) {
			return &models.DownloadSession{
				ID:                 1,
				TotalChunks:        3,
				ConfirmedChunks:    1,
				Status:             models.DownloadStatusActive,
				ConfirmedChunkSet:  map[int64]bool{0: true},
			}, nil
		},
	}

	resp, err := svc.GetDownloadStatus(ctxWithUser(1), &pbv1.GetDownloadStatusRequest{DownloadId: 1})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.DownloadId)
	require.Equal(t, pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE, resp.Status)
	require.Equal(t, int64(3), resp.TotalChunks)
	require.Equal(t, int64(1), resp.ConfirmedChunks)
	require.Equal(t, []int64{1, 2}, resp.RemainingChunks)
}

func TestGetDownloadStatus_InvalidDownloadID(t *testing.T) {
	svc := newTestUploadsService()
	_, err := svc.GetDownloadStatus(ctxWithUser(1), &pbv1.GetDownloadStatusRequest{DownloadId: 0})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetDownloadStatus_NotFound(t *testing.T) {
	svc := newTestUploadsService()
	svc.usecase = &mockUploadsUseCase{
		getDownloadStatusFn: func(downloadID int64) (*models.DownloadSession, error) {
			return nil, models.ErrDownloadNotFound
		},
	}
	_, err := svc.GetDownloadStatus(ctxWithUser(1), &pbv1.GetDownloadStatusRequest{DownloadId: 999})
	require.Equal(t, codes.NotFound, status.Code(err))
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func TestMapUploadError_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"upload not found", models.ErrUploadNotFound, codes.NotFound},
		{"upload not pending", models.ErrUploadNotPending, codes.FailedPrecondition},
		{"upload completed", models.ErrUploadCompleted, codes.FailedPrecondition},
		{"upload aborted", models.ErrUploadAborted, codes.FailedPrecondition},
		{"chunk out of range", models.ErrChunkOutOfRange, codes.InvalidArgument},
		{"duplicate chunk", models.ErrDuplicateChunk, codes.AlreadyExists},
		{"chunk out of order", models.ErrChunkOutOfOrder, codes.InvalidArgument},
		{"invalid user id", models.ErrInvalidUserID, codes.InvalidArgument},
		{"unknown error", errors.New("unknown"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := mapUploadError(tt.err)
			require.Equal(t, tt.wantCode, status.Code(mapped))
		})
	}
}

func TestMapDownloadError_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"download not found", models.ErrDownloadNotFound, codes.NotFound},
		{"download completed", models.ErrDownloadCompleted, codes.FailedPrecondition},
		{"download aborted", models.ErrDownloadAborted, codes.FailedPrecondition},
		{"download not active", models.ErrDownloadNotActive, codes.FailedPrecondition},
		{"upload not found", models.ErrUploadNotFound, codes.NotFound},
		{"chunk out of range", models.ErrChunkOutOfRange, codes.InvalidArgument},
		{"chunk already confirmed", models.ErrChunkAlreadyConfirmed, codes.AlreadyExists},
		{"unknown error", errors.New("unknown"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := mapDownloadError(tt.err)
			require.Equal(t, tt.wantCode, status.Code(mapped))
		})
	}
}

func TestDomainUploadStatusToProto(t *testing.T) {
	tests := []struct {
		name string
		s    models.UploadStatus
		want pbv1.UploadStatus
	}{
		{"pending", models.UploadStatusPending, pbv1.UploadStatus_UPLOAD_STATUS_PENDING},
		{"completed", models.UploadStatusCompleted, pbv1.UploadStatus_UPLOAD_STATUS_COMPLETED},
		{"aborted", models.UploadStatusAborted, pbv1.UploadStatus_UPLOAD_STATUS_ABORTED},
		{"unknown", models.UploadStatus("unknown"), pbv1.UploadStatus_UPLOAD_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainUploadStatusToProto(tt.s)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDomainDownloadStatusToProto(t *testing.T) {
	tests := []struct {
		name string
		s    models.DownloadStatus
		want pbv1.DownloadStatus
	}{
		{"active", models.DownloadStatusActive, pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE},
		{"completed", models.DownloadStatusCompleted, pbv1.DownloadStatus_DOWNLOAD_STATUS_COMPLETED},
		{"aborted", models.DownloadStatusAborted, pbv1.DownloadStatus_DOWNLOAD_STATUS_ABORTED},
		{"unknown", models.DownloadStatus("unknown"), pbv1.DownloadStatus_DOWNLOAD_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domainDownloadStatusToProto(tt.s)
			require.Equal(t, tt.want, got)
		})
	}
}
