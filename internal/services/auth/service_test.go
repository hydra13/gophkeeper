package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
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
	if _, exists := m.byEmail[user.Email]; exists {
		return models.ErrEmailAlreadyExists
	}
	m.nextID++
	user.ID = m.nextID
	copy := *user
	m.byEmail[user.Email] = &copy
	m.byID[user.ID] = &copy
	return nil
}

func (m *memUserRepo) GetUserByEmail(email string) (*models.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	copy := *user
	return &copy, nil
}

func (m *memUserRepo) GetUserByID(id int64) (*models.User, error) {
	user, ok := m.byID[id]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	copy := *user
	return &copy, nil
}

type memSessionRepo struct {
	sessions []*models.Session
	nextID   int64
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{
		sessions: make([]*models.Session, 0),
		nextID:   0,
	}
}

func (m *memSessionRepo) CreateSession(session *models.Session) error {
	m.nextID++
	session.ID = m.nextID
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *memSessionRepo) GetSession(id int64) (*models.Session, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, models.ErrSessionExpired
}

func (m *memSessionRepo) GetSessionByRefreshToken(token string) (*models.Session, error) {
	for _, s := range m.sessions {
		if s.RefreshToken == token {
			return s, nil
		}
	}
	return nil, models.ErrSessionExpired
}

func (m *memSessionRepo) RevokeSession(id int64) error {
	now := time.Now()
	for _, s := range m.sessions {
		if s.ID == id {
			s.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (m *memSessionRepo) RevokeSessionsByUser(userID int64) error {
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID {
			s.RevokedAt = &now
		}
	}
	return nil
}

func (m *memSessionRepo) UpdateLastSeenAt(id int64) error {
	now := time.Now()
	for _, s := range m.sessions {
		if s.ID == id {
			s.LastSeenAt = now
			return nil
		}
	}
	return nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	jwtManager, err := NewJWTManager("test-secret-key-for-tests", 15*time.Minute)
	require.NoError(t, err)

	svc, err := NewService(
		newMemUserRepo(),
		newMemSessionRepo(),
		jwtManager,
		time.Hour,
	)
	require.NoError(t, err)
	return svc
}

func TestRegister_Success(t *testing.T) {
	svc := newTestService(t)

	userID, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)
	require.NotZero(t, userID)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "password1")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "user@example.com", "password2")
	require.ErrorIs(t, err, models.ErrEmailAlreadyExists)
}

func TestLogin_Success(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	accessToken, refreshToken, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	_, _, err = svc.Login(context.Background(), "user@example.com", "wrong-password", "device-1", "MacBook", "cli")
	require.ErrorIs(t, err, models.ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := newTestService(t)

	_, _, err := svc.Login(context.Background(), "nonexistent@example.com", "password", "device-1", "MacBook", "cli")
	require.ErrorIs(t, err, models.ErrInvalidCredentials)
}

func TestValidateToken_Valid(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	accessToken, _, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	userID, err := svc.ValidateToken(accessToken)
	require.NoError(t, err)
	require.Equal(t, int64(1), userID)
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.ValidateToken("invalid-token")
	require.Error(t, err)
}

func TestValidateToken_Expired(t *testing.T) {
	jwtManager, err := NewJWTManager("test-secret", 1*time.Nanosecond)
	require.NoError(t, err)

	svc, err := NewService(newMemUserRepo(), newMemSessionRepo(), jwtManager, time.Hour)
	require.NoError(t, err)

	token, err := svc.jwt.NewAccessToken(1, 1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = svc.ValidateToken(token)
	require.Error(t, err)
}

func TestRefresh_Success(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	_, refreshToken, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	newAccess, newRefresh, err := svc.Refresh(context.Background(), refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, newAccess)
	require.NotEmpty(t, newRefresh)
	require.NotEqual(t, refreshToken, newRefresh)

	// Новый access-токен валиден
	userID, err := svc.ValidateToken(newAccess)
	require.NoError(t, err)
	require.Equal(t, int64(1), userID)
}

func TestRefresh_InvalidRefreshToken(t *testing.T) {
	svc := newTestService(t)

	_, _, err := svc.Refresh(context.Background(), "nonexistent-refresh-token")
	require.ErrorIs(t, err, models.ErrUnauthorized)
}

func TestRefresh_RevokedSession(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	_, refreshToken, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	// Первый refresh ок
	_, _, err = svc.Refresh(context.Background(), refreshToken)
	require.NoError(t, err)

	// Повторный refresh тем же токеном — сессия уже отозвана
	_, _, err = svc.Refresh(context.Background(), refreshToken)
	require.ErrorIs(t, err, models.ErrSessionRevoked)
}

func TestLogout_Success(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	accessToken, refreshToken, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)

	err = svc.Logout(context.Background(), accessToken)
	require.NoError(t, err)

	// После logout refresh-токен не работает (сессия отозвана)
	_, _, err = svc.Refresh(context.Background(), refreshToken)
	require.Error(t, err)
}

func TestLogout_InvalidToken(t *testing.T) {
	svc := newTestService(t)

	err := svc.Logout(context.Background(), "invalid-access-token")
	require.ErrorIs(t, err, models.ErrUnauthorized)
}

func TestFullAuthFlow(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// 1. Register
	userID, err := svc.Register(ctx, "test@example.com", "my-password")
	require.NoError(t, err)
	require.NotZero(t, userID)

	// 2. Login
	accessToken, refreshToken, err := svc.Login(ctx, "test@example.com", "my-password", "device-1", "Laptop", "cli")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// 3. Validate access token
	validatedID, err := svc.ValidateToken(accessToken)
	require.NoError(t, err)
	require.Equal(t, userID, validatedID)

	// 4. Refresh
	newAccess, newRefresh, err := svc.Refresh(ctx, refreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, newAccess)
	require.NotEmpty(t, newRefresh)

	// 5. Validate new access token
	validatedID, err = svc.ValidateToken(newAccess)
	require.NoError(t, err)
	require.Equal(t, userID, validatedID)

	// 6. Old refresh token is revoked
	_, _, err = svc.Refresh(ctx, refreshToken)
	require.ErrorIs(t, err, models.ErrSessionRevoked)

	// 7. Logout
	err = svc.Logout(ctx, newAccess)
	require.NoError(t, err)

	// 8. After logout, refresh is revoked
	_, _, err = svc.Refresh(ctx, newRefresh)
	require.Error(t, err)
}

func TestValidateSession_Success(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	accessToken, _, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	userID, err := svc.ValidateSession(accessToken)
	require.NoError(t, err)
	require.Equal(t, int64(1), userID)
}

func TestValidateSession_RevokedAfterLogout(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Register(context.Background(), "user@example.com", "super-secret")
	require.NoError(t, err)

	accessToken, _, err := svc.Login(context.Background(), "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	// Token works before logout
	userID, err := svc.ValidateSession(accessToken)
	require.NoError(t, err)
	require.Equal(t, int64(1), userID)

	// Logout
	err = svc.Logout(context.Background(), accessToken)
	require.NoError(t, err)

	// Same access token is rejected after logout — session revoked
	_, err = svc.ValidateSession(accessToken)
	require.ErrorIs(t, err, models.ErrSessionRevoked)
}

func TestValidateSession_InvalidToken(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.ValidateSession("invalid-token")
	require.Error(t, err)
}

func TestNewJWTManager_EmptySecret(t *testing.T) {
	_, err := NewJWTManager("", 15*time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "secret is required")
}

func TestNewJWTManager_DefaultTTL(t *testing.T) {
	mgr, err := NewJWTManager("secret", 0)
	require.NoError(t, err)
	require.Equal(t, defaultAccessTTL, mgr.accessTTL)
}

func TestLogoutAllDevices(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "super-secret")
	require.NoError(t, err)

	// Login on device-1
	access1, refresh1, err := svc.Login(ctx, "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	// Login on device-2
	access2, refresh2, err := svc.Login(ctx, "user@example.com", "super-secret", "device-2", "iPhone", "cli")
	require.NoError(t, err)

	// Logout all devices
	err = svc.LogoutAllDevices(ctx, 1)
	require.NoError(t, err)

	// Both sessions should be revoked
	_, err = svc.ValidateSession(access1)
	require.ErrorIs(t, err, models.ErrSessionRevoked)

	_, err = svc.ValidateSession(access2)
	require.ErrorIs(t, err, models.ErrSessionRevoked)

	// Both refresh tokens should fail
	_, _, err = svc.Refresh(ctx, refresh1)
	require.ErrorIs(t, err, models.ErrSessionRevoked)

	_, _, err = svc.Refresh(ctx, refresh2)
	require.ErrorIs(t, err, models.ErrSessionRevoked)
}

func TestNewService_NilDeps(t *testing.T) {
	jwtManager, err := NewJWTManager("test-secret", 15*time.Minute)
	require.NoError(t, err)

	_, err = NewService(nil, newMemSessionRepo(), jwtManager, time.Hour)
	require.Error(t, err)

	_, err = NewService(newMemUserRepo(), nil, jwtManager, time.Hour)
	require.Error(t, err)

	_, err = NewService(newMemUserRepo(), newMemSessionRepo(), nil, time.Hour)
	require.Error(t, err)
}

func TestLogout_RevokesSpecificSession(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "super-secret")
	require.NoError(t, err)

	// Login on device-1
	access1, refresh1, err := svc.Login(ctx, "user@example.com", "super-secret", "device-1", "MacBook", "cli")
	require.NoError(t, err)

	// Login on device-2
	access2, refresh2, err := svc.Login(ctx, "user@example.com", "super-secret", "device-2", "iPhone", "cli")
	require.NoError(t, err)

	// Logout from device-1 only revokes session-1
	err = svc.Logout(ctx, access1)
	require.NoError(t, err)

	// device-1 tokens no longer work
	_, err = svc.ValidateSession(access1)
	require.ErrorIs(t, err, models.ErrSessionRevoked)
	_, _, err = svc.Refresh(ctx, refresh1)
	require.ErrorIs(t, err, models.ErrSessionRevoked)

	// device-2 still works
	userID, err := svc.ValidateSession(access2)
	require.NoError(t, err)
	require.Equal(t, int64(1), userID)
	_, newRefresh, err := svc.Refresh(ctx, refresh2)
	require.NoError(t, err)
	require.NotEmpty(t, newRefresh)
}
