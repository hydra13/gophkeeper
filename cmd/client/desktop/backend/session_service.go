package backend

import (
	"context"
	"time"

	"github.com/hydra13/gophkeeper/pkg/clientcore"
)

const (
	clientName = "desktop-client"
	clientType = "desktop"
)

type SessionService struct {
	core *clientcore.ClientCore
	info AppInfo
}

func NewSessionService(core *clientcore.ClientCore, info AppInfo) *SessionService {
	return &SessionService{
		core: core,
		info: info,
	}
}

func (s *SessionService) GetSessionState() SessionState {
	auth, _ := s.core.CurrentAuth()
	return SessionState{
		Authenticated: s.core.IsAuthenticated(),
		Email:         auth.Email,
		DeviceID:      auth.DeviceID,
		AppName:       s.info.AppName,
		Version:       s.info.Version,
		ServerAddress: s.info.ServerAddress,
		CacheDir:      s.info.CacheDir,
	}
}

func (s *SessionService) Login(email, password string) (SessionState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.core.LoginWithClient(ctx, email, password, clientName, clientType); err != nil {
		return SessionState{}, normalizeError(err)
	}

	return s.GetSessionState(), nil
}

func (s *SessionService) Register(email, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return normalizeError(s.core.Register(ctx, email, password))
}

func (s *SessionService) Logout() (SessionState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.core.Logout(ctx); err != nil {
		return SessionState{}, normalizeError(err)
	}

	return s.GetSessionState(), nil
}
