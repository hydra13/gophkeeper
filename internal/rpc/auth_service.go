package rpc

import (
	"context"

	"github.com/hydra13/gophkeeper/internal/rpc/pbv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthService gRPC имплементация.
type AuthService struct {
	pbv1.UnimplementedAuthServiceServer
}

// NewAuthService создаёт заглушку AuthService.
func NewAuthService() *AuthService {
	return &AuthService{}
}

func (s *AuthService) Register(context.Context, *pbv1.RegisterRequest) (*pbv1.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "auth service not implemented")
}

func (s *AuthService) Login(context.Context, *pbv1.LoginRequest) (*pbv1.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "auth service not implemented")
}

func (s *AuthService) Refresh(context.Context, *pbv1.RefreshRequest) (*pbv1.RefreshResponse, error) {
	return nil, status.Error(codes.Unimplemented, "auth service not implemented")
}

func (s *AuthService) Logout(context.Context, *pbv1.LogoutRequest) (*pbv1.LogoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "auth service not implemented")
}
