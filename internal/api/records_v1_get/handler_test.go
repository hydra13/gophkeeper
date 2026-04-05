package records_v1_get

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

	"github.com/hydra13/gophkeeper/internal/api/records_v1_get/mocks"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		userID    int64
		setupMock func(mc *minimock.Controller) RecordService
		wantCode  int
		wantLen   int
	}{
		{
			name:   "success empty list",
			userID: 1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					ListRecordsMock.Expect(int64(1), models.RecordType(""), false).Return([]models.Record{}, nil)
			},
			wantCode: http.StatusOK,
			wantLen:  0,
		},
		{
			name:   "success with records",
			userID: 1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					ListRecordsMock.Expect(int64(1), models.RecordType(""), false).Return([]models.Record{
					{
						ID:         1,
						UserID:     1,
						Type:       models.RecordTypeLogin,
						Name:       "Test",
						Revision:   1,
						DeviceID:   "dev-1",
						KeyVersion: 1,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
					{
						ID:         2,
						UserID:     1,
						Type:       models.RecordTypeText,
						Name:       "Note",
						Revision:   2,
						DeviceID:   "dev-1",
						KeyVersion: 1,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
				}, nil)
			},
			wantCode: http.StatusOK,
			wantLen:  2,
		},
		{
			name:   "success soft deleted record",
			userID: 1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					ListRecordsMock.Expect(int64(1), models.RecordType(""), false).Return([]models.Record{
					{
						ID:         1,
						UserID:     1,
						Type:       models.RecordTypeLogin,
						Name:       "Deleted",
						Revision:   2,
						DeviceID:   "dev-1",
						KeyVersion: 1,
						DeletedAt:  &now,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
				}, nil)
			},
			wantCode: http.StatusOK,
			wantLen:  1,
		},
		{
			name:   "unauthorized",
			userID: 0,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:   "service error",
			userID: 1,
			setupMock: func(mc *minimock.Controller) RecordService {
				return mocks.NewRecordServiceMock(mc).
					ListRecordsMock.Expect(int64(1), models.RecordType(""), false).Return(nil, errors.New("db error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			handler := NewHandler(tt.setupMock(mc))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
			if tt.userID > 0 {
				req = req.WithContext(middlewares.ContextWithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			require.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			if tt.wantCode != http.StatusOK {
				return
			}

			var resp ListRecordsResponse
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Len(t, resp.Records, tt.wantLen)
		})
	}
}
