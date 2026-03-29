package middlewares

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hydra13/gophkeeper/internal/models"
)

func TestUnaryAuth_PublicMethod(t *testing.T) {
	validator := &mockValidator{}
	allowMethods := map[string]struct{}{
		"/gophkeeper.v1.AuthService/Register": {},
	}

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Register"}
	_, err := UnaryAuth(validator, allowMethods)(context.Background(), nil, info, handler)
	require.NoError(t, err)
	require.True(t, called)
}

func TestUnaryAuth_NoToken(t *testing.T) {
	validator := &mockValidator{}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(context.Background(), nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuth_ValidSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 42, nil
		},
	}
	allowMethods := map[string]struct{}{}

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		userID, ok := UserIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, int64(42), userID)
		return "ok", nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.NoError(t, err)
	require.True(t, called)
}

func TestUnaryAuth_RevokedSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionRevoked
		},
	}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer revoked-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuth_ExpiredSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionExpired
		},
	}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer expired-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
