package auth_login_v1_post

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

	"github.com/hydra13/gophkeeper/internal/api/auth_login_v1_post/mocks"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func(mc *minimock.Controller) UserService
		wantStatus int
		wantErr    bool
	}{
		{
			name: "Success",
			body: LoginRequest{Email: "user@example.com", Password: "password123", DeviceID: "device-1", DeviceName: "MacBook", ClientType: "cli"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					LoginMock.
					Expect(minimock.AnyContext, "user@example.com", "password123", "device-1", "MacBook", "cli").
					Return("access-token", "refresh-token", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Invalid body",
			body: "not-json",
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty email",
			body: LoginRequest{Email: "", Password: "password123", DeviceID: "device-1"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty password",
			body: LoginRequest{Email: "user@example.com", Password: "", DeviceID: "device-1"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty device_id",
			body: LoginRequest{Email: "user@example.com", Password: "password123"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Invalid credentials",
			body: LoginRequest{Email: "user@example.com", Password: "wrong", DeviceID: "device-1"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					LoginMock.
					Expect(minimock.AnyContext, "user@example.com", "wrong", "device-1", "", "").
					Return("", "", models.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "User not found",
			body: LoginRequest{Email: "unknown@example.com", Password: "password123", DeviceID: "device-1"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					LoginMock.
					Expect(minimock.AnyContext, "unknown@example.com", "password123", "device-1", "", "").
					Return("", "", models.ErrUserNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "Internal error",
			body: LoginRequest{Email: "user@example.com", Password: "password123", DeviceID: "device-1"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					LoginMock.
					Expect(minimock.AnyContext, "user@example.com", "password123", "device-1", "", "").
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
			userService := tt.setupMock(mc)

			handler := NewHandler(userService, zerolog.Nop())

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			resp := rec.Result()
			t.Cleanup(func() {
				require.NoError(t, resp.Body.Close())
			})

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if !tt.wantErr {
				var response LoginResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "access-token", response.AccessToken)
				assert.Equal(t, "refresh-token", response.RefreshToken)
				return
			}

			require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			var response map[string]string
			err := json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)
			require.NotEmpty(t, response["error"])
		})
	}
}
