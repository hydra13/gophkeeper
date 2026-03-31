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

// SessionService инкапсулирует операции входа, регистрации и выхода.
type SessionService struct {
	core *clientcore.ClientCore
	info AppInfo
}

// NewSessionService создает сервис для операций с пользовательской сессией.
func NewSessionService(core *clientcore.ClientCore, info AppInfo) *SessionService {
	return &SessionService{
		core: core,
		info: info,
	}
}

// GetSessionState возвращает текущее состояние авторизации и сведения о клиенте.
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

// Login выполняет вход и возвращает обновленное состояние сессии.
func (s *SessionService) Login(email, password string) (SessionState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.core.LoginWithClient(ctx, email, password, clientName, clientType); err != nil {
		return SessionState{}, normalizeError(err)
	}

	return s.GetSessionState(), nil
}

// Register регистрирует нового пользователя.
func (s *SessionService) Register(email, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return normalizeError(s.core.Register(ctx, email, password))
}

// Logout завершает сессию и возвращает ее новое состояние.
func (s *SessionService) Logout() (SessionState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.core.Logout(ctx); err != nil {
		return SessionState{}, normalizeError(err)
	}

	return s.GetSessionState(), nil
}
