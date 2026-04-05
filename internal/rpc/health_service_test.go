package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

func TestHealthCheck_OK(t *testing.T) {
	svc := NewHealthService()

	resp, err := svc.HealthCheck(context.Background(), &pbv1.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, pbv1.HealthStatus_HEALTH_STATUS_OK, resp.Status)
}
