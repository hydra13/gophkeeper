//go:generate minimock -i .UserRepo,.SessionRepo,.TokenManager -o mocks -s _mock.go -g
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/services/passwords"
)

const defaultSessionTTL = 24 * time.Hour

type UserRepo interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
}

type SessionRepo interface {
	CreateSession(session *models.Session) error
	GetSession(id int64) (*models.Session, error)
	GetSessionByRefreshToken(token string) (*models.Session, error)
	RevokeSession(id int64) error
	RevokeSessionsByUser(userID int64) error
	UpdateLastSeenAt(id int64) error
}

type TokenManager interface {
	NewRefreshToken() (string, error)
	NewAccessToken(userID int64, sessionID int64) (string, error)
	ValidateToken(token string) (int64, int64, error)
}

// Service реализует полный auth use-case: register, login, refresh, logout, logout-all-devices.
type Service struct {
	users      UserRepo
	sessions   SessionRepo
	jwt        TokenManager
	sessionTTL time.Duration
	now        func() time.Time
}

// NewService создаёт новый auth service.
func NewService(
	usersRepo UserRepo,
	sessionsRepo SessionRepo,
	jwtManager TokenManager,
	sessionTTL time.Duration,
) (*Service, error) {
	if usersRepo == nil {
		return nil, errors.New("user repository is required")
	}
	if sessionsRepo == nil {
		return nil, errors.New("session repository is required")
	}
	if jwtManager == nil {
		return nil, errors.New("jwt manager is required")
	}
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}
	return &Service{
		users:      usersRepo,
		sessions:   sessionsRepo,
		jwt:        jwtManager,
		sessionTTL: sessionTTL,
		now:        time.Now,
	}, nil
}

// Register регистрирует нового пользователя.
func (s *Service) Register(ctx context.Context, email, password string) (int64, error) {
	hash, err := passwords.HashPassword(password)
	if err != nil {
		return 0, err
	}
	user := &models.User{
		Email:        email,
		PasswordHash: hash,
	}
	if err := s.users.CreateUser(user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

// Login проверяет учётные данные, создаёт сессию и возвращает пару токенов.
func (s *Service) Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (string, string, error) {
	user, err := s.users.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return "", "", models.ErrInvalidCredentials
		}
		return "", "", err
	}
	if err := passwords.ComparePassword(user.PasswordHash, password); err != nil {
		return "", "", models.ErrInvalidCredentials
	}

	refreshToken, err := s.jwt.NewRefreshToken()
	if err != nil {
		return "", "", err
	}

	now := s.now()
	session := &models.Session{
		UserID:       user.ID,
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		ClientType:   clientType,
		RefreshToken: refreshToken,
		LastSeenAt:   now,
		ExpiresAt:    now.Add(s.sessionTTL),
	}
	if err := s.sessions.CreateSession(session); err != nil {
		return "", "", err
	}

	accessToken, err := s.jwt.NewAccessToken(user.ID, session.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// Refresh обновляет пару токенов по refresh-токену.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	session, err := s.sessions.GetSessionByRefreshToken(refreshToken)
	if err != nil {
		return "", "", models.ErrUnauthorized
	}
	if !session.IsActive() {
		if session.IsExpired() {
			return "", "", models.ErrSessionExpired
		}
		return "", "", models.ErrSessionRevoked
	}

	// Отзываем старую сессию.
	if err := s.sessions.RevokeSession(session.ID); err != nil {
		return "", "", err
	}

	newRefreshToken, err := s.jwt.NewRefreshToken()
	if err != nil {
		return "", "", err
	}

	now := s.now()
	newSession := &models.Session{
		UserID:       session.UserID,
		DeviceID:     session.DeviceID,
		DeviceName:   session.DeviceName,
		ClientType:   session.ClientType,
		RefreshToken: newRefreshToken,
		LastSeenAt:   now,
		ExpiresAt:    now.Add(s.sessionTTL),
	}
	if err := s.sessions.CreateSession(newSession); err != nil {
		return "", "", err
	}

	accessToken, err := s.jwt.NewAccessToken(session.UserID, newSession.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

// Logout отзывает сессию по access-токену.
func (s *Service) Logout(ctx context.Context, accessToken string) error {
	_, sessionID, err := s.jwt.ValidateToken(accessToken)
	if err != nil {
		return models.ErrUnauthorized
	}
	return s.sessions.RevokeSession(sessionID)
}

// LogoutAllDevices отзывает все сессии пользователя.
func (s *Service) LogoutAllDevices(ctx context.Context, userID int64) error {
	return s.sessions.RevokeSessionsByUser(userID)
}

// ValidateToken проверяет access-токен и возвращает userID.
func (s *Service) ValidateToken(token string) (int64, error) {
	userID, _, err := s.jwt.ValidateToken(token)
	return userID, err
}

// ValidateSession проверяет access-токен, валидирует серверную сессию
// и обновляет last_seen_at. Возвращает userID.
func (s *Service) ValidateSession(token string) (int64, error) {
	userID, sessionID, err := s.jwt.ValidateToken(token)
	if err != nil {
		return 0, err
	}
	session, err := s.sessions.GetSession(sessionID)
	if err != nil {
		return 0, models.ErrUnauthorized
	}
	if session.IsRevoked() {
		return 0, models.ErrSessionRevoked
	}
	if session.IsExpired() {
		return 0, models.ErrSessionExpired
	}
	if err := s.sessions.UpdateLastSeenAt(sessionID); err != nil {
		return 0, err
	}
	return userID, nil
}
