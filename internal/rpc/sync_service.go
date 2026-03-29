package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SyncService gRPC имплементация.
type SyncService struct {
	pbv1.UnimplementedSyncServiceServer
}

// NewSyncService создаёт заглушку SyncService.
func NewSyncService() *SyncService {
	return &SyncService{}
}

func (s *SyncService) Push(context.Context, *pbv1.PushRequest) (*pbv1.PushResponse, error) {
	return nil, status.Error(codes.Unimplemented, "sync service not implemented")
}

func (s *SyncService) Pull(context.Context, *pbv1.PullRequest) (*pbv1.PullResponse, error) {
	return nil, status.Error(codes.Unimplemented, "sync service not implemented")
}
