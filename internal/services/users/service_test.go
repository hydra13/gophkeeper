package users

import (
	"context"
	"errors"
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

// --- Token generator tests (randomTokenGenerator, generateToken) ---

func TestGenerateToken(t *testing.T) {
	t.Run("returns non-empty base64url string", func(t *testing.T) {
		token, err := generateToken()
		require.NoError(t, err)
		require.NotEmpty(t, token)
		// 32 bytes -> base64 raw url = 43 chars
		require.Len(t, token, 43)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		tokens := make(map[string]struct{}, 100)
		for i := 0; i < 100; i++ {
			token, err := generateToken()
			require.NoError(t, err)
			tokens[token] = struct{}{}
		}
		require.Len(t, tokens, 100, "all generated tokens must be unique")
	})
}

func TestRandomTokenGeneratorNewAccessToken(t *testing.T) {
	gen := randomTokenGenerator{}

	t.Run("returns valid token", func(t *testing.T) {
		token, err := gen.NewAccessToken(42)
		require.NoError(t, err)
		require.NotEmpty(t, token)
	})

	t.Run("ignores userID but still works", func(t *testing.T) {
		token1, err1 := gen.NewAccessToken(1)
		token2, err2 := gen.NewAccessToken(999)
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotEqual(t, token1, token2, "tokens should differ across calls")
	})
}

func TestRandomTokenGeneratorNewRefreshToken(t *testing.T) {
	gen := randomTokenGenerator{}

	t.Run("returns valid token", func(t *testing.T) {
		token, err := gen.NewRefreshToken()
		require.NoError(t, err)
		require.NotEmpty(t, token)
	})

	t.Run("generates unique refresh tokens", func(t *testing.T) {
		token1, err1 := gen.NewRefreshToken()
		token2, err2 := gen.NewRefreshToken()
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotEqual(t, token1, token2)
	})
}

// --- NewService edge cases ---

func TestNewServiceDefaults(t *testing.T) {
	t.Run("nil token generator defaults to randomTokenGenerator", func(t *testing.T) {
		svc, err := NewService(newMemUserRepo(), &memSessionRepo{}, nil, time.Hour)
		require.NoError(t, err)
		require.NotNil(t, svc)
		require.NotNil(t, svc.tokens)
	})

	t.Run("zero or negative session TTL defaults to 24h", func(t *testing.T) {
		svc, err := NewService(newMemUserRepo(), &memSessionRepo{}, nil, 0)
		require.NoError(t, err)
		require.Equal(t, defaultSessionTTL, svc.sessionTTL)
	})

	t.Run("negative session TTL defaults to 24h", func(t *testing.T) {
		svc, err := NewService(newMemUserRepo(), &memSessionRepo{}, nil, -5*time.Minute)
		require.NoError(t, err)
		require.Equal(t, defaultSessionTTL, svc.sessionTTL)
	})

	t.Run("nil user repo returns error", func(t *testing.T) {
		_, err := NewService(nil, &memSessionRepo{}, nil, time.Hour)
		require.Error(t, err)
		require.Contains(t, err.Error(), "user repository")
	})

	t.Run("nil session repo returns error", func(t *testing.T) {
		_, err := NewService(newMemUserRepo(), nil, nil, time.Hour)
		require.Error(t, err)
		require.Contains(t, err.Error(), "session repository")
	})
}

// --- Login edge cases ---

func TestServiceLoginUserNotFound(t *testing.T) {
	userRepo := newMemUserRepo()
	sessionRepo := &memSessionRepo{}
	svc, err := NewService(userRepo, sessionRepo, fixedTokenGenerator{}, time.Hour)
	require.NoError(t, err)

	_, _, err = svc.Login(context.Background(), "nonexistent@example.com", "pass", "dev1", "Phone", "cli")
	require.ErrorIs(t, err, models.ErrInvalidCredentials)
}

type errTokenGenerator struct {
	accessErr  error
	refreshErr error
}

func (e errTokenGenerator) NewAccessToken(_ int64) (string, error) {
	return "", e.accessErr
}

func (e errTokenGenerator) NewRefreshToken() (string, error) {
	return "", e.refreshErr
}

func TestServiceLoginTokenErrors(t *testing.T) {
	userRepo := newMemUserRepo()
	hash, err := passwords.HashPassword("pass")
	require.NoError(t, err)
	require.NoError(t, userRepo.CreateUser(&models.User{Email: "a@b.c", PasswordHash: hash}))

	t.Run("access token generation error", func(t *testing.T) {
		svc, err := NewService(userRepo, &memSessionRepo{}, errTokenGenerator{accessErr: errors.New("access fail")}, time.Hour)
		require.NoError(t, err)

		_, _, err = svc.Login(context.Background(), "a@b.c", "pass", "d", "n", "c")
		require.EqualError(t, err, "access fail")
	})

	t.Run("refresh token generation error", func(t *testing.T) {
		svc, err := NewService(userRepo, &memSessionRepo{}, errTokenGenerator{refreshErr: errors.New("refresh fail")}, time.Hour)
		require.NoError(t, err)

		_, _, err = svc.Login(context.Background(), "a@b.c", "pass", "d", "n", "c")
		require.EqualError(t, err, "refresh fail")
	})
}
