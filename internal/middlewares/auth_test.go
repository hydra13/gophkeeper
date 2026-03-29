package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/models"
)

type mockValidator struct {
	validateSessionFn func(token string) (int64, error)
}

func (m *mockValidator) ValidateSession(token string) (int64, error) {
	return m.validateSessionFn(token)
}

func TestAuth_NoHeader(t *testing.T) {
	handler := Auth(&mockValidator{}, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidFormat(t *testing.T) {
	handler := Auth(&mockValidator{}, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Basic abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_EmptyToken(t *testing.T) {
	handler := Auth(&mockValidator{}, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ValidSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 42, nil
		},
	}

	var called bool
	handler := Auth(validator, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		userID, ok := UserIDFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, int64(42), userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAuth_RevokedSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionRevoked
		},
	}

	handler := Auth(validator, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Bearer revoked-session-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ExpiredSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionExpired
		},
	}

	handler := Auth(validator, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Bearer expired-session-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, errors.New("invalid token")
		},
	}

	handler := Auth(validator, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
