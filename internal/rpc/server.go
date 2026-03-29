package rpc

import (
	"google.golang.org/grpc"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// Server gRPC сервер.
type Server struct {
	auth    pbv1.AuthServiceServer
	data    pbv1.DataServiceServer
	sync    pbv1.SyncServiceServer
	uploads pbv1.UploadsServiceServer
	health  pbv1.HealthServiceServer
}

// NewServer создаёт gRPC сервер с зависимостями.
func NewServer(
	auth pbv1.AuthServiceServer,
	data pbv1.DataServiceServer,
	sync pbv1.SyncServiceServer,
	uploads pbv1.UploadsServiceServer,
	health pbv1.HealthServiceServer,
) *Server {
	return &Server{
		auth:    auth,
		data:    data,
		sync:    sync,
		uploads: uploads,
		health:  health,
	}
}

// Register регистрирует сервисы в gRPC сервере.
func (s *Server) Register(g *grpc.Server) {
	pbv1.RegisterAuthServiceServer(g, s.auth)
	pbv1.RegisterDataServiceServer(g, s.data)
	pbv1.RegisterSyncServiceServer(g, s.sync)
	pbv1.RegisterUploadsServiceServer(g, s.uploads)
	pbv1.RegisterHealthServiceServer(g, s.health)
}
