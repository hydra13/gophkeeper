package records_by_id_v1_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/records_by_id_v1_get/mocks"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loginPayload := models.LoginPayload{Login: "user", Password: "pass"}

	tests := []struct {
		name      string
		recordID  string
		userID    int64
		setupMock func(mc *minimock.Controller) RecordService
		wantCode  int
	}{
		{
			name:     "success",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					GetRecordMock.Expect(int64(1)).Return(&models.Record{
					ID:         1,
					UserID:     1,
					Type:       models.RecordTypeLogin,
					Name:       "Test",
					Revision:   1,
					DeviceID:   "dev-1",
					KeyVersion: 1,
					Payload:    loginPayload,
					CreatedAt:  now,
					UpdatedAt:  now,
				}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			recordID: "1",
			userID:   0,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid record id",
			recordID: "abc",
			userID:   1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "record not found",
			recordID: "999",
			userID:   1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					GetRecordMock.Expect(int64(999)).Return(nil, models.ErrRecordNotFound)
			},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "access denied",
			recordID: "1",
			userID:   2,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					GetRecordMock.Expect(int64(1)).Return(&models.Record{
					ID:         1,
					UserID:     1,
					Type:       models.RecordTypeLogin,
					Name:       "Test",
					DeviceID:   "dev-1",
					KeyVersion: 1,
					Payload:    loginPayload,
					CreatedAt:  now,
					UpdatedAt:  now,
				}, nil)
			},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "internal error",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					GetRecordMock.Expect(int64(1)).Return(nil, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			handler := NewHandler(tt.setupMock(mc))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/records/{id}", nil)
			req.SetPathValue("id", tt.recordID)
			if tt.userID > 0 {
				req = req.WithContext(middlewares.ContextWithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			assert.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
		})
	}
}

func TestRecordToResponse(t *testing.T) {
	t.Parallel()

	now := time.Now()
	resp := recordToResponse(&models.Record{
		ID:         1,
		UserID:     1,
		Type:       models.RecordTypeText,
		Name:       "note",
		Revision:   2,
		DeviceID:   "dev-1",
		KeyVersion: 1,
		Payload:    models.TextPayload{Content: "hello"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	body, err := json.Marshal(resp)
	require.NoError(t, err)
	require.Contains(t, string(body), `"record"`)
	require.Contains(t, string(body), `"name":"note"`)
}
