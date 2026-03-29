package auth_refresh_v1_post

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/auth_refresh_v1_post/mocks"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(mc *minimock.Controller) TokenService
		wantStatus int
		wantErr    bool
	}{
		{
			name: "Success",
			body: RefreshRequest{RefreshToken: "valid-refresh-token"},
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					RefreshMock.
					Expect(minimock.AnyContext, "valid-refresh-token").
					Return("new-access-token", "new-refresh-token", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid body",
			body:       "not-json",
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty refresh token",
			body: RefreshRequest{RefreshToken: ""},
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Session expired",
			body: RefreshRequest{RefreshToken: "expired-token"},
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					RefreshMock.
					Expect(minimock.AnyContext, "expired-token").
					Return("", "", models.ErrSessionExpired)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "Session revoked",
			body: RefreshRequest{RefreshToken: "revoked-token"},
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					RefreshMock.
					Expect(minimock.AnyContext, "revoked-token").
					Return("", "", models.ErrSessionRevoked)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "Internal error",
			body: RefreshRequest{RefreshToken: "valid-token"},
			setupMock: func(mc *minimock.Controller) TokenService {
				return mocks.NewTokenServiceMock(mc).
					RefreshMock.
					Expect(minimock.AnyContext, "valid-token").
					Return("", "", context.DeadlineExceeded)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			tokenService := tt.setupMock(mc)

			handler := NewHandler(tokenService, zerolog.Nop())

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if !tt.wantErr {
				var response RefreshResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "new-access-token", response.AccessToken)
				assert.Equal(t, "new-refresh-token", response.RefreshToken)
			}
		})
	}
}
