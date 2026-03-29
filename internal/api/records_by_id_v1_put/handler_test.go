package recordsbyidv1put

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

func newActiveRecord() *models.Record {
	now := time.Now()
	return &models.Record{
		ID: 1, UserID: 1, Type: models.RecordTypeLogin,
		Name: "Test", Revision: 1, DeviceID: "dev-1",
		KeyVersion: 1, Payload: models.LoginPayload{Login: "user", Password: "pass"},
		CreatedAt: now, UpdatedAt: now,
	}
}

func newDeletedRecord() *models.Record {
	now := time.Now()
	r := newActiveRecord()
	r.DeletedAt = &now
	return r
}

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name      string
		recordID  string
		userID    int64
		body      interface{}
		existing  *models.Record
		getErr    error
		updateErr error
		wantCode  int
	}{
		{
			name:     "success",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name:       "Updated",
				Revision:   2,
				DeviceID:   "dev-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "newuser", Password: "newpass"},
			},
			existing: newActiveRecord(),
			wantCode: http.StatusOK,
		},
		{
			name:     "unauthorized",
			recordID: "1",
			userID:   0,
			body:     UpdateRecordRequest{},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid record id",
			recordID: "abc",
			userID:   1,
			body:     UpdateRecordRequest{},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "record not found",
			recordID: "999",
			userID:   1,
			body:     UpdateRecordRequest{},
			getErr:   models.ErrRecordNotFound,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "access denied - wrong user",
			recordID: "1",
			userID:   2,
			body:     UpdateRecordRequest{},
			existing: newActiveRecord(),
			wantCode: http.StatusForbidden,
		},
		{
			name:     "deleted record cannot be updated",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name: "Updated", Revision: 2, DeviceID: "dev-1", KeyVersion: 1,
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			existing: newDeletedRecord(),
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "revision conflict - same revision",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name: "Updated", Revision: 1, DeviceID: "dev-1", KeyVersion: 1,
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			existing: newActiveRecord(),
			wantCode: http.StatusConflict,
		},
		{
			name:     "revision conflict - lower revision",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name: "Updated", Revision: 0, DeviceID: "dev-1", KeyVersion: 1,
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			existing: newActiveRecord(),
			wantCode: http.StatusConflict,
		},
		{
			name:     "service update conflict",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name: "Updated", Revision: 2, DeviceID: "dev-1", KeyVersion: 1,
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			existing:  newActiveRecord(),
			updateErr: models.ErrRevisionConflict,
			wantCode:  http.StatusConflict,
		},
		{
			name:     "service internal error on get",
			recordID: "1",
			userID:   1,
			body:     UpdateRecordRequest{},
			getErr:   errors.New("db error"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "service internal error on update",
			recordID: "1",
			userID:   1,
			body: UpdateRecordRequest{
				Name: "Updated", Revision: 2, DeviceID: "dev-1", KeyVersion: 1,
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			existing:  newActiveRecord(),
			updateErr: errors.New("db error"),
			wantCode:  http.StatusInternalServerError,
		},
		{
			name:     "invalid JSON body",
			recordID: "1",
			userID:   1,
			body:     "not-json{{{",
			existing: newActiveRecord(),
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &recordServiceMock{
				record:    tt.existing,
				getErr:    tt.getErr,
				updateErr: tt.updateErr,
			}
			handler := NewHandler(mock)

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/records/{id}", bytes.NewReader(bodyBytes))
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
	updateErr error
}

func (m *recordServiceMock) GetRecord(id int64) (*models.Record, error) {
	return m.record, m.getErr
}

func (m *recordServiceMock) UpdateRecord(record *models.Record) error {
	return m.updateErr
}
