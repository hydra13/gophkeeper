package recordsbyidv1get

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	now := time.Now()
	loginPayload := models.LoginPayload{Login: "user", Password: "pass"}

	tests := []struct {
		name        string
		recordID    string
		userID      int64
		serviceErr  error
		serviceResp *models.Record
		wantCode    int
	}{
		{
			name:     "success",
			recordID: "1",
			userID:   1,
			serviceResp: &models.Record{
				ID: 1, UserID: 1, Type: models.RecordTypeLogin,
				Name: "Test", Revision: 1, DeviceID: "dev-1",
				KeyVersion: 1, Payload: loginPayload,
				CreatedAt: now, UpdatedAt: now,
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			recordID: "1",
			userID:   0,
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid record id",
			recordID: "abc",
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "record not found",
			recordID:   "999",
			userID:     1,
			serviceErr: models.ErrRecordNotFound,
			wantCode:   http.StatusNotFound,
		},
		{
			name:     "access denied - wrong user",
			recordID: "1",
			userID:   2,
			serviceResp: &models.Record{
				ID: 1, UserID: 1, Type: models.RecordTypeLogin,
				Name: "Test", DeviceID: "dev-1",
				KeyVersion: 1, Payload: loginPayload,
				CreatedAt: now, UpdatedAt: now,
			},
			wantCode: http.StatusForbidden,
		},
		{
			name:       "internal error",
			recordID:   "1",
			userID:     1,
			serviceErr: errors.New("db error"),
			wantCode:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordServiceMock{
				record: tt.serviceResp,
				err:    tt.serviceErr,
			}
			handler := NewHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/records/{id}", nil)
			req.SetPathValue("id", tt.recordID)
			if tt.userID > 0 {
				ctx := context.WithValue(req.Context(), userIDKey{}, tt.userID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			handler.Handle(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("got status %d, want %d; body: %s", rec.Code, tt.wantCode, rec.Body.String())
			}
		})
	}
}

type recordServiceMock struct {
	record *models.Record
	err    error
}

func (m *recordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.err
}
