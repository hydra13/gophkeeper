package auth_logout_v1_post

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/hydra13/gophkeeper/api/auth_logout_v1_post/mocks"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		authHeader string
		setupMock  func(mc *minimock.Controller) TokenService
		wantStatus int
	}{
		{
			name:       "Success",
			authHeader: "Bearer valid-access-token",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					LogoutMock.
					Expect(minimock.AnyContext, "valid-access-token").
					Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "Missing authorization header",
			authHeader: "",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Invalid authorization format",
			authHeader: "Basic dXNlcjpwYXNz",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Session expired",
			authHeader: "Bearer expired-token",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					LogoutMock.
					Expect(minimock.AnyContext, "expired-token").
					Return(models.ErrSessionExpired)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Unauthorized",
			authHeader: "Bearer invalid-token",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					LogoutMock.
					Expect(minimock.AnyContext, "invalid-token").
					Return(models.ErrUnauthorized)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Internal error",
			authHeader: "Bearer valid-token",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					LogoutMock.
					Expect(minimock.AnyContext, "valid-token").
					Return(context.DeadlineExceeded)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			tokenService := tt.setupMock(mc)

			handler := NewHandler(tokenService, zerolog.Nop())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(nil))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
