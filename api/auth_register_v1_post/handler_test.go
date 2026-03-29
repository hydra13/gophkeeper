package auth_register_v1_post

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

	"github.com/hydra13/gophkeeper/api/auth_register_v1_post/mocks"
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
			body: RegisterRequest{Email: "user@example.com", Password: "password123"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					RegisterMock.
					Expect(minimock.AnyContext, "user@example.com", "password123").
					Return(int64(1), nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Invalid body",
			body:       "not-json",
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty email",
			body: RegisterRequest{Email: "", Password: "password123"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Empty password",
			body: RegisterRequest{Email: "user@example.com", Password: ""},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Short password",
			body: RegisterRequest{Email: "user@example.com", Password: "1234567"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc)
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "Email already exists",
			body: RegisterRequest{Email: "existing@example.com", Password: "password123"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					RegisterMock.
					Expect(minimock.AnyContext, "existing@example.com", "password123").
					Return(int64(0), models.ErrEmailAlreadyExists)
			},
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
		{
			name: "Internal error",
			body: RegisterRequest{Email: "user@example.com", Password: "password123"},
			setupMock: func(mc *minimock.Controller) UserService {
				return mocks.NewUserServiceMock(mc).
					RegisterMock.
					Expect(minimock.AnyContext, "user@example.com", "password123").
					Return(int64(0), context.DeadlineExceeded)
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

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if !tt.wantErr {
				var response RegisterResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, int64(1), response.UserID)
			}
		})
	}
}
