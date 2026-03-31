package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// HealthService реализует gRPC health-check.
type HealthService struct {
	pbv1.UnimplementedHealthServiceServer
}

// NewHealthService создаёт HealthService.
func NewHealthService() *HealthService {
	return &HealthService{}
}

// HealthCheck возвращает статус доступности сервиса.
func (s *HealthService) HealthCheck(context.Context, *pbv1.HealthCheckRequest) (*pbv1.HealthCheckResponse, error) {
	return &pbv1.HealthCheckResponse{
		Status: pbv1.HealthStatus_HEALTH_STATUS_OK,
	}, nil
}
