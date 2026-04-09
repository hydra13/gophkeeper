package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// HealthService отвечает на проверки доступности сервиса.
type HealthService struct {
	pbv1.UnimplementedHealthServiceServer
}

// NewHealthService создаёт HealthService.
func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) HealthCheck(context.Context, *pbv1.HealthCheckRequest) (*pbv1.HealthCheckResponse, error) {
	return &pbv1.HealthCheckResponse{
		Status: pbv1.HealthStatus_HEALTH_STATUS_OK,
	}, nil
}
