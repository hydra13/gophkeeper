package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UploadsService gRPC имплементация chunk upload/download.
type UploadsService struct {
	pbv1.UnimplementedUploadsServiceServer
}

// NewUploadsService создаёт заглушку UploadsService.
func NewUploadsService() *UploadsService {
	return &UploadsService{}
}

func (s *UploadsService) CreateUploadSession(context.Context, *pbv1.CreateUploadSessionRequest) (*pbv1.CreateUploadSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) UploadChunk(grpc.ClientStreamingServer[pbv1.UploadChunkRequest, pbv1.UploadChunkResponse]) error {
	return status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) GetUploadStatus(context.Context, *pbv1.GetUploadStatusRequest) (*pbv1.GetUploadStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) CreateDownloadSession(context.Context, *pbv1.CreateDownloadSessionRequest) (*pbv1.CreateDownloadSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) DownloadChunk(*pbv1.DownloadChunkRequest, grpc.ServerStreamingServer[pbv1.DownloadChunkResponse]) error {
	return status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) ConfirmChunk(context.Context, *pbv1.ConfirmChunkRequest) (*pbv1.ConfirmChunkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "uploads service not implemented")
}

func (s *UploadsService) GetDownloadStatus(context.Context, *pbv1.GetDownloadStatusRequest) (*pbv1.GetDownloadStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "uploads service not implemented")
}
