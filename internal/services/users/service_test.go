package users

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/hydra13/gophkeeper/internal/services/passwords"
)

type memUserRepo struct {
	nextID  int64
	byEmail map[string]*models.User
	byID    map[int64]*models.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{
		nextID:  0,
		byEmail: make(map[string]*models.User),
		byID:    make(map[int64]*models.User),
	}
}

func (m *memUserRepo) CreateUser(user *models.User) error {
	if user == nil {
		return nil
	}
	if _, exists := m.byEmail[user.Email]; exists {
		return models.ErrEmailAlreadyExists
	}
	m.nextID++
	user.ID = m.nextID
	copyUser := *user
	m.byEmail[user.Email] = &copyUser
	m.byID[user.ID] = &copyUser
	return nil
}

func (m *memUserRepo) GetUserByEmail(email string) (*models.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (m *memUserRepo) GetUserByID(id int64) (*models.User, error) {
	user, ok := m.byID[id]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

type memSessionRepo struct {
	sessions []*models.Session
}

func (m *memSessionRepo) CreateSession(session *models.Session) error {
	if session == nil {
		return nil
	}
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *memSessionRepo) GetSession(id int64) (*models.Session, error) {
	return nil, models.ErrSessionExpired
}

func (m *memSessionRepo) GetSessionByRefreshToken(token string) (*models.Session, error) {
	return nil, models.ErrSessionExpired
}

func (m *memSessionRepo) RevokeSession(id int64) error {
	return nil
}

func (m *memSessionRepo) RevokeSessionsByUser(userID int64) error {
	return nil
}

func (m *memSessionRepo) UpdateLastSeenAt(id int64) error {
	return nil
}

type fixedTokenGenerator struct {
	access  string
	refresh string
}

func (f fixedTokenGenerator) NewAccessToken(userID int64) (string, error) {
	return f.access, nil
}

func (f fixedTokenGenerator) NewRefreshToken() (string, error) {
	return f.refresh, nil
}

func TestServiceRegisterHashesPassword(t *testing.T) {
	userRepo := newMemUserRepo()
	sessionRepo := &memSessionRepo{}
	service, err := NewService(userRepo, sessionRepo, fixedTokenGenerator{access: "a", refresh: "r"}, time.Hour)
	require.NoError(t, err)

	userID, err := service.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)
	require.NotZero(t, userID)

	stored, err := userRepo.GetUserByEmail("user@example.com")
	require.NoError(t, err)
	require.NotEqual(t, "super-secret", stored.PasswordHash)
	require.NoError(t, passwords.ComparePassword(stored.PasswordHash, "super-secret"))
}

func TestServiceLoginValidatesPassword(t *testing.T) {
	userRepo := newMemUserRepo()
	sessionRepo := &memSessionRepo{}
	service, err := NewService(userRepo, sessionRepo, fixedTokenGenerator{access: "access-token", refresh: "refresh-token"}, time.Hour)
	require.NoError(t, err)

	hash, err := passwords.HashPassword("super-secret")
	require.NoError(t, err)

	err = userRepo.CreateUser(&models.User{
		Email:        "user@example.com",
		PasswordHash: hash,
	})
	require.NoError(t, err)

	access, refresh, err := service.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)
	require.Equal(t, "access-token", access)
	require.Equal(t, "refresh-token", refresh)
	require.Len(t, sessionRepo.sessions, 1)
	require.Equal(t, "refresh-token", sessionRepo.sessions[0].RefreshToken)

	_, _, err = service.Login(context.Background(), "user@example.com", "wrong-password", "device-1", "MacBook", "cli")
	require.ErrorIs(t, err, models.ErrInvalidCredentials)
}
