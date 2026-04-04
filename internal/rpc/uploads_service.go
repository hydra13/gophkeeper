package rpc

import (
	"context"
	"errors"
	"io"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UploadsUseCase interface {
	CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
	GetUploadSessionByID(uploadID int64) (*models.UploadSession, error)
	UploadChunk(uploadID, chunkIndex int64, data []byte) (received, total int64, completed bool, missing []int64, err error)
	CreateDownloadSession(userID, recordID int64) (*models.DownloadSession, error)
	DownloadChunkByID(downloadID, chunkIndex int64) (*models.Chunk, error)
	ConfirmChunk(downloadID, chunkIndex int64) (confirmed, total int64, status models.DownloadStatus, err error)
	GetDownloadStatus(downloadID int64) (*models.DownloadSession, error)
}

// UploadsService реализует gRPC-ручки работы с бинарными payload.
type UploadsService struct {
	pbv1.UnimplementedUploadsServiceServer
	usecase UploadsUseCase
	log     zerolog.Logger
}

// NewUploadsService создаёт UploadsService.
func NewUploadsService(usecase UploadsUseCase, log zerolog.Logger) *UploadsService {
	return &UploadsService{
		usecase: usecase,
		log:     log,
	}
}

func (s *UploadsService) CreateUploadSession(ctx context.Context, req *pbv1.CreateUploadSessionRequest) (*pbv1.CreateUploadSessionResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.RecordId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "record_id is required")
	}
	if req.TotalChunks <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total_chunks must be positive")
	}
	if req.ChunkSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chunk_size must be positive")
	}
	if req.TotalSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total_size must be positive")
	}

	uploadID, err := s.usecase.CreateSession(userID, req.RecordId, req.TotalChunks, req.ChunkSize, req.TotalSize, req.KeyVersion)
	if err != nil {
		return nil, mapUploadError(err)
	}

	return &pbv1.CreateUploadSessionResponse{
		UploadId: uploadID,
		Status:   pbv1.UploadStatus_UPLOAD_STATUS_PENDING,
	}, nil
}

func (s *UploadsService) UploadChunk(stream grpc.ClientStreamingServer[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse]) error {
	ctx := stream.Context()

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return err
	}
	_ = userID

	var received, total int64
	var completed bool

	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return status.Error(codes.Canceled, "client canceled upload")
			}
			if errors.Is(err, io.EOF) {
				break
			}
			return mapUploadError(err)
		}

		if req.UploadId <= 0 {
			return status.Error(codes.InvalidArgument, "upload_id is required")
		}
		if len(req.Data) == 0 {
			return status.Error(codes.InvalidArgument, "data is required")
		}

		received, total, completed, _, err = s.usecase.UploadChunk(req.UploadId, req.ChunkIndex, req.Data)
		if err != nil {
			return mapUploadError(err)
		}
	}

	resp := &pbv1.UploadChunkResponse{
		ReceivedChunks: received,
		TotalChunks:    total,
	}

	if completed {
		resp.Status = pbv1.UploadStatus_UPLOAD_STATUS_COMPLETED
	} else {
		resp.Status = pbv1.UploadStatus_UPLOAD_STATUS_PENDING
	}

	if err := stream.SendAndClose(resp); err != nil {
		return mapUploadError(err)
	}

	return nil
}

func (s *UploadsService) GetUploadStatus(ctx context.Context, req *pbv1.GetUploadStatusRequest) (*pbv1.GetUploadStatusResponse, error) {
	if _, err := userIDFromContext(ctx); err != nil {
		return nil, err
	}

	if req.UploadId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required")
	}

	session, err := s.usecase.GetUploadSessionByID(req.UploadId)
	if err != nil {
		return nil, mapUploadError(err)
	}

	return &pbv1.GetUploadStatusResponse{
		UploadId:       session.ID,
		Status:         domainUploadStatusToProto(session.Status),
		TotalChunks:    session.TotalChunks,
		ReceivedChunks: session.ReceivedChunks,
		MissingChunks:  session.MissingChunks(),
	}, nil
}

func (s *UploadsService) CreateDownloadSession(ctx context.Context, req *pbv1.CreateDownloadSessionRequest) (*pbv1.CreateDownloadSessionResponse, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.RecordId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "record_id is required")
	}

	download, err := s.usecase.CreateDownloadSession(userID, req.RecordId)
	if err != nil {
		return nil, mapDownloadError(err)
	}

	return &pbv1.CreateDownloadSessionResponse{
		DownloadId:  download.ID,
		Status:      pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE,
		TotalChunks: download.TotalChunks,
	}, nil
}

func (s *UploadsService) DownloadChunk(req *pbv1.DownloadChunkRequest, stream grpc.ServerStreamingServer[pbv1.DownloadChunkResponse]) error {
	ctx := stream.Context()

	if _, err := userIDFromContext(ctx); err != nil {
		return err
	}

	if req.DownloadId <= 0 {
		return status.Error(codes.InvalidArgument, "download_id is required")
	}

	download, err := s.usecase.GetDownloadStatus(req.DownloadId)
	if err != nil {
		return mapDownloadError(err)
	}

	var chunkIndices []int64
	if req.ChunkIndex >= 0 {
		chunkIndices = []int64{req.ChunkIndex}
	} else {
		chunkIndices = download.RemainingChunks()
	}

	for _, idx := range chunkIndices {
		chunk, err := s.usecase.DownloadChunkByID(req.DownloadId, idx)
		if err != nil {
			return mapDownloadError(err)
		}

		if err := stream.Send(&pbv1.DownloadChunkResponse{
			DownloadId: req.DownloadId,
			ChunkIndex: chunk.ChunkIndex,
			Data:       chunk.Data,
		}); err != nil {
			return mapDownloadError(err)
		}
	}

	return nil
}

func (s *UploadsService) ConfirmChunk(ctx context.Context, req *pbv1.ConfirmChunkRequest) (*pbv1.ConfirmChunkResponse, error) {
	if _, err := userIDFromContext(ctx); err != nil {
		return nil, err
	}

	if req.DownloadId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "download_id is required")
	}
	if req.ChunkIndex < 0 {
		return nil, status.Error(codes.InvalidArgument, "chunk_index must be non-negative")
	}

	confirmed, total, dlStatus, err := s.usecase.ConfirmChunk(req.DownloadId, req.ChunkIndex)
	if err != nil {
		return nil, mapDownloadError(err)
	}

	return &pbv1.ConfirmChunkResponse{
		Status:          domainDownloadStatusToProto(dlStatus),
		ConfirmedChunks: confirmed,
		TotalChunks:     total,
	}, nil
}

func (s *UploadsService) GetDownloadStatus(ctx context.Context, req *pbv1.GetDownloadStatusRequest) (*pbv1.GetDownloadStatusResponse, error) {
	if _, err := userIDFromContext(ctx); err != nil {
		return nil, err
	}

	if req.DownloadId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "download_id is required")
	}

	download, err := s.usecase.GetDownloadStatus(req.DownloadId)
	if err != nil {
		return nil, mapDownloadError(err)
	}

	return &pbv1.GetDownloadStatusResponse{
		DownloadId:      download.ID,
		Status:          domainDownloadStatusToProto(download.Status),
		TotalChunks:     download.TotalChunks,
		ConfirmedChunks: download.ConfirmedChunks,
		RemainingChunks: download.RemainingChunks(),
	}, nil
}

func domainUploadStatusToProto(s models.UploadStatus) pbv1.UploadStatus {
	switch s {
	case models.UploadStatusPending:
		return pbv1.UploadStatus_UPLOAD_STATUS_PENDING
	case models.UploadStatusCompleted:
		return pbv1.UploadStatus_UPLOAD_STATUS_COMPLETED
	case models.UploadStatusAborted:
		return pbv1.UploadStatus_UPLOAD_STATUS_ABORTED
	default:
		return pbv1.UploadStatus_UPLOAD_STATUS_UNSPECIFIED
	}
}

func domainDownloadStatusToProto(s models.DownloadStatus) pbv1.DownloadStatus {
	switch s {
	case models.DownloadStatusActive:
		return pbv1.DownloadStatus_DOWNLOAD_STATUS_ACTIVE
	case models.DownloadStatusCompleted:
		return pbv1.DownloadStatus_DOWNLOAD_STATUS_COMPLETED
	case models.DownloadStatusAborted:
		return pbv1.DownloadStatus_DOWNLOAD_STATUS_ABORTED
	default:
		return pbv1.DownloadStatus_DOWNLOAD_STATUS_UNSPECIFIED
	}
}

func mapUploadError(err error) error {
	switch {
	case errors.Is(err, models.ErrUploadNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrUploadNotPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrUploadCompleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrUploadAborted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrChunkOutOfRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrDuplicateChunk):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, models.ErrChunkOutOfOrder):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrInvalidUserID):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func mapDownloadError(err error) error {
	switch {
	case errors.Is(err, models.ErrDownloadNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrDownloadCompleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrDownloadAborted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrDownloadNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, models.ErrUploadNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, models.ErrChunkOutOfRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, models.ErrChunkAlreadyConfirmed):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, models.ErrChunkOutOfOrder):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
