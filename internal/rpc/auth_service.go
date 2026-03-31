package rpc

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthUseCase определяет auth-операции для gRPC слоя.
type AuthUseCase interface {
	Register(ctx context.Context, email, password string) (int64, error)
	Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, accessToken string) error
}

// AuthService реализует gRPC-ручки аутентификации.
type AuthService struct {
	pbv1.UnimplementedAuthServiceServer
	auth AuthUseCase
	log  zerolog.Logger
}

// NewAuthService создаёт AuthService с зависимостями.
func NewAuthService(auth AuthUseCase, log zerolog.Logger) *AuthService {
	return &AuthService{
		auth: auth,
		log:  log,
	}
}

// Register регистрирует пользователя по email и паролю.
func (s *AuthService) Register(ctx context.Context, req *pbv1.RegisterRequest) (*pbv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	if len(req.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	userID, err := s.auth.Register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return &pbv1.RegisterResponse{UserId: userID}, nil
}

// Login аутентифицирует пользователя и возвращает access и refresh токены.
func (s *AuthService) Login(ctx context.Context, req *pbv1.LoginRequest) (*pbv1.LoginResponse, error) {
	if req.Email == "" || req.Password == "" || req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password and device_id are required")
	}

	accessToken, refreshToken, err := s.auth.Login(ctx, req.Email, req.Password, req.DeviceId, req.DeviceName, req.ClientType)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return &pbv1.LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Refresh обновляет access и refresh токены по refresh token.
func (s *AuthService) Refresh(ctx context.Context, req *pbv1.RefreshRequest) (*pbv1.RefreshResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	accessToken, refreshToken, err := s.auth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, mapAuthError(err)
	}
	return &pbv1.RefreshResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// Logout завершает текущую сессию по access token из metadata.
func (s *AuthService) Logout(ctx context.Context, req *pbv1.LogoutRequest) (*pbv1.LogoutResponse, error) {
	token := extractGRPCToken(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "authorization required")
	}

	if err := s.auth.Logout(ctx, token); err != nil {
		return nil, mapAuthError(err)
	}
	return &pbv1.LogoutResponse{}, nil
}

func extractGRPCToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}

func mapAuthError(err error) error {
	switch {
	case errors.Is(err, models.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrUserNotFound):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, models.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, models.ErrSessionExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrSessionRevoked):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
