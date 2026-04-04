//go:generate minimock -i .UserRepo,.SessionRepo,.TokenGenerator -o mocks -s _mock.go -g
package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
}

type TokenGenerator interface {
	NewAccessToken(userID int64) (string, error)
	NewRefreshToken() (string, error)
}

// Service реализует регистрацию и вход пользователя.
type Service struct {
	users      UserRepo
	sessions   SessionRepo
	tokens     TokenGenerator
	sessionTTL time.Duration
	now        func() time.Time
}

// NewService создаёт сервис пользователей.
func NewService(usersRepo UserRepo, sessionsRepo SessionRepo, tokens TokenGenerator, sessionTTL time.Duration) (*Service, error) {
	if usersRepo == nil {
		return nil, errors.New("user repository is required")
	}
	if sessionsRepo == nil {
		return nil, errors.New("session repository is required")
	}
	if tokens == nil {
		tokens = randomTokenGenerator{}
	}
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}
	return &Service{
		users:      usersRepo,
		sessions:   sessionsRepo,
		tokens:     tokens,
		sessionTTL: sessionTTL,
		now:        time.Now,
	}, nil
}

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
	accessToken, err := s.tokens.NewAccessToken(user.ID)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := s.tokens.NewRefreshToken()
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
	return accessToken, refreshToken, nil
}

type randomTokenGenerator struct{}

func (randomTokenGenerator) NewAccessToken(userID int64) (string, error) {
	return generateToken()
}

func (randomTokenGenerator) NewRefreshToken() (string, error) {
	return generateToken()
}

func generateToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}
