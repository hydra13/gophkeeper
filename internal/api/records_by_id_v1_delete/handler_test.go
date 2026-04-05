package records_by_id_v1_delete

import (
	"bytes"
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
	loginPayload := models.LoginPayload{Login: "user", Password: "pass"}

	activeRecord := &models.Record{
		ID: 1, UserID: 1, Type: models.RecordTypeLogin,
		Name: "Test", Revision: 1, DeviceID: "dev-1",
		KeyVersion: 1, Payload: loginPayload,
		CreatedAt: now, UpdatedAt: now,
	}

	deletedRecord := &models.Record{
		ID: 1, UserID: 1, Type: models.RecordTypeLogin,
		Name: "Test", Revision: 1, DeviceID: "dev-1",
		KeyVersion: 1, Payload: loginPayload,
		DeletedAt: &now, CreatedAt: now, UpdatedAt: now,
	}

	tests := []struct {
		name       string
		recordID   string
		userID     int64
		deviceID   string
		serviceErr error
		record     *models.Record
		deleteErr  error
		wantCode   int
	}{
		{
			name:     "success",
			recordID: "1",
			userID:   1,
			deviceID: "dev-1",
			record:   activeRecord,
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			recordID: "1",
			userID:   0,
			deviceID: "dev-1",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid record id",
			recordID: "abc",
			userID:   1,
			deviceID: "dev-1",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing device_id",
			recordID: "1",
			userID:   1,
			deviceID: "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:       "record not found - idempotent",
			recordID:   "999",
			userID:     1,
			deviceID:   "dev-1",
			serviceErr: models.ErrRecordNotFound,
			wantCode:   http.StatusOK,
		},
		{
			name:     "access denied - wrong user",
			recordID: "1",
			userID:   2,
			deviceID: "dev-1",
			record:   activeRecord,
			wantCode: http.StatusForbidden,
		},
		{
			name:     "already deleted - idempotent",
			recordID: "1",
			userID:   1,
			deviceID: "dev-1",
			record:   deletedRecord,
			wantCode: http.StatusOK,
		},
		{
			name:       "internal error on get",
			recordID:   "1",
			userID:     1,
			deviceID:   "dev-1",
			serviceErr: errors.New("db error"),
			wantCode:   http.StatusInternalServerError,
		},
		{
			name:      "internal error on delete",
			recordID:  "1",
			userID:    1,
			deviceID:  "dev-1",
			record:    activeRecord,
			deleteErr: errors.New("db error"),
			wantCode:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordServiceMock{
				record:    tt.record,
				getErr:    tt.serviceErr,
				deleteErr: tt.deleteErr,
			}
			handler := NewHandler(mock)

			var body []byte
			if tt.recordID != "abc" && tt.userID != 0 {
				body, _ = json.Marshal(DeleteRecordRequest{DeviceID: tt.deviceID})
			} else {
				body, _ = json.Marshal(DeleteRecordRequest{DeviceID: tt.deviceID})
			}
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/records/{id}", bytes.NewReader(body))
			req.SetPathValue("id", tt.recordID)
			if tt.userID > 0 {
				ctx := middlewares.ContextWithUserID(req.Context(), tt.userID)
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
	record    *models.Record
	getErr    error
	deleteErr error
}

func (m *recordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.getErr
}

func (m *recordServiceMock) DeleteRecord(id int64, deviceID string) error {
	return m.deleteErr
}
