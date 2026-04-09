package health_v1_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/health_v1_get/mocks"
	"github.com/hydra13/gophkeeper/internal/api/responses"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		setupMock  func(mc *minimock.Controller) HealthChecker
		wantCode   int
		wantStatus string
	}{
		{
			name:   "ok",
			method: http.MethodGet,
			setupMock: func(mc *minimock.Controller) HealthChecker {
				return mocks.NewHealthCheckerMock(mc).
					HealthMock.Return(nil)
			},
			wantCode:   http.StatusOK,
			wantStatus: "ok",
		},
		{
			name:   "error",
			method: http.MethodGet,
			setupMock: func(mc *minimock.Controller) HealthChecker {
				return mocks.NewHealthCheckerMock(mc).
					HealthMock.Return(errors.New("db unavailable"))
			},
			wantCode:   http.StatusOK,
			wantStatus: "error",
		},
		{
			name:   "method not allowed",
			method: http.MethodPost,
			setupMock: func(mc *minimock.Controller) HealthChecker {
				return mocks.NewHealthCheckerMock(mc)
			},
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			handler := NewHandler(tt.setupMock(mc))

			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			if tt.wantStatus == "" {
				var resp responses.ErrorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, "method not allowed", resp.Error)
				return
			}

			var resp Response
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, tt.wantStatus, resp.Status)
		})
	}
}
