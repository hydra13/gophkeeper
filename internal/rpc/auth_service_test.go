package rpc

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
)

// mockAuthUseCase реализует AuthUseCase для тестов gRPC.
type mockAuthUseCase struct {
	registerFn      func(ctx context.Context, email, password string) (int64, error)
	loginFn         func(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error)
	refreshFn       func(ctx context.Context, refreshToken string) (string, string, error)
	logoutFn        func(ctx context.Context, accessToken string) error
}

func (m *mockAuthUseCase) Register(ctx context.Context, email, password string) (int64, error) {
	return m.registerFn(ctx, email, password)
}

func (m *mockAuthUseCase) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	return m.loginFn(ctx, email, password, deviceID, deviceName, clientType)
}

func (m *mockAuthUseCase) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	return m.refreshFn(ctx, refreshToken)
}

func (m *mockAuthUseCase) Logout(ctx context.Context, accessToken string) error {
	return m.logoutFn(ctx, accessToken)
}

func newTestAuthService() *AuthService {
	return NewAuthService(&mockAuthUseCase{}, zerolog.Nop())
}

func TestRegister_Success(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		registerFn: func(_ context.Context, email, password string) (int64, error) {
			return 1, nil
		},
	}

	resp, err := svc.Register(context.Background(), &pbv1.RegisterRequest{
		Email: "user@example.com", Password: "password123",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.UserId)
}

func TestRegister_EmptyEmail(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Register(context.Background(), &pbv1.RegisterRequest{
		Email: "", Password: "password123",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Register(context.Background(), &pbv1.RegisterRequest{
		Email: "user@example.com", Password: "",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Register(context.Background(), &pbv1.RegisterRequest{
		Email: "user@example.com", Password: "1234567",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		registerFn: func(_ context.Context, _, _ string) (int64, error) {
			return 0, models.ErrEmailAlreadyExists
		},
	}

	_, err := svc.Register(context.Background(), &pbv1.RegisterRequest{
		Email: "user@example.com", Password: "password123",
	})
	require.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestLogin_Success(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		loginFn: func(_ context.Context, _, _, _, _, _ string) (string, string, error) {
			return "access-token", "refresh-token", nil
		},
	}

	resp, err := svc.Login(context.Background(), &pbv1.LoginRequest{
		Email: "user@example.com", Password: "password", DeviceId: "device-1",
	})
	require.NoError(t, err)
	require.Equal(t, "access-token", resp.AccessToken)
	require.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestLogin_MissingFields(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Login(context.Background(), &pbv1.LoginRequest{
		Email: "", Password: "password", DeviceId: "device-1",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = svc.Login(context.Background(), &pbv1.LoginRequest{
		Email: "user@example.com", Password: "", DeviceId: "device-1",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = svc.Login(context.Background(), &pbv1.LoginRequest{
		Email: "user@example.com", Password: "password", DeviceId: "",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		loginFn: func(_ context.Context, _, _, _, _, _ string) (string, string, error) {
			return "", "", models.ErrInvalidCredentials
		},
	}

	_, err := svc.Login(context.Background(), &pbv1.LoginRequest{
		Email: "user@example.com", Password: "wrong", DeviceId: "device-1",
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRefresh_Success(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		refreshFn: func(_ context.Context, _ string) (string, string, error) {
			return "new-access", "new-refresh", nil
		},
	}

	resp, err := svc.Refresh(context.Background(), &pbv1.RefreshRequest{
		RefreshToken: "valid-token",
	})
	require.NoError(t, err)
	require.Equal(t, "new-access", resp.AccessToken)
	require.Equal(t, "new-refresh", resp.RefreshToken)
}

func TestRefresh_EmptyToken(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Refresh(context.Background(), &pbv1.RefreshRequest{
		RefreshToken: "",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRefresh_SessionExpired(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		refreshFn: func(_ context.Context, _ string) (string, string, error) {
			return "", "", models.ErrSessionExpired
		},
	}

	_, err := svc.Refresh(context.Background(), &pbv1.RefreshRequest{
		RefreshToken: "expired-token",
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRefresh_SessionRevoked(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		refreshFn: func(_ context.Context, _ string) (string, string, error) {
			return "", "", models.ErrSessionRevoked
		},
	}

	_, err := svc.Refresh(context.Background(), &pbv1.RefreshRequest{
		RefreshToken: "revoked-token",
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogout_Success(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		logoutFn: func(_ context.Context, _ string) error {
			return nil
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access-token"))
	resp, err := svc.Logout(ctx, &pbv1.LogoutRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestLogout_NoToken(t *testing.T) {
	svc := newTestAuthService()
	_, err := svc.Logout(context.Background(), &pbv1.LogoutRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogout_InvalidToken(t *testing.T) {
	svc := newTestAuthService()
	svc.auth = &mockAuthUseCase{
		logoutFn: func(_ context.Context, _ string) error {
			return models.ErrUnauthorized
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer bad-token"))
	_, err := svc.Logout(ctx, &pbv1.LogoutRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
