package recordsv1get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		userID      int64
		serviceErr  error
		serviceResp []models.Record
		wantCode    int
		wantLen     int
	}{
		{
			name:     "success empty list",
			userID:   1,
			wantCode: http.StatusOK,
			wantLen:  0,
		},
		{
			name:   "success with records",
			userID: 1,
			serviceResp: []models.Record{
				{
					ID: 1, UserID: 1, Type: models.RecordTypeLogin,
					Name: "Test", Revision: 1, DeviceID: "dev-1",
					KeyVersion: 1, CreatedAt: now, UpdatedAt: now,
				},
				{
					ID: 2, UserID: 1, Type: models.RecordTypeText,
					Name: "Note", Revision: 2, DeviceID: "dev-1",
					KeyVersion: 1, CreatedAt: now, UpdatedAt: now,
				},
			},
			wantCode: http.StatusOK,
			wantLen:  2,
		},
		{
			name:   "success soft deleted record",
			userID: 1,
			serviceResp: []models.Record{
				{
					ID: 1, UserID: 1, Type: models.RecordTypeLogin,
					Name: "Deleted", Revision: 2, DeviceID: "dev-1",
					KeyVersion: 1, DeletedAt: &now, CreatedAt: now, UpdatedAt: now,
				},
			},
			wantCode: http.StatusOK,
			wantLen:  1,
		},
		{
			name:     "unauthorized - no user",
			userID:   0,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:       "service error",
			userID:     1,
			serviceErr: errors.New("db error"),
			wantCode:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordServiceMock{
				records: tt.serviceResp,
				err:     tt.serviceErr,
			}
			handler := NewHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
			if tt.userID > 0 {
				ctx := middlewares.ContextWithUserID(req.Context(), tt.userID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got status %d, want %d; body: %s", rec.Code, tt.wantCode, rec.Body.String())
			}

			if rec.Code == http.StatusOK {
				var resp ListRecordsResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp.Records) != tt.wantLen {
					t.Errorf("got %d records, want %d", len(resp.Records), tt.wantLen)
				}
			}
		})
	}
}

type recordServiceMock struct {
	records []models.Record
	err     error
}

func (m *recordServiceMock) ListRecords(userID int64, recordType models.RecordType, includeDeleted bool) ([]models.Record, error) {
	return m.records, m.err
}
