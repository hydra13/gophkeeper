package recordsv1post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		userID     int64
		serviceErr error
		wantCode   int
	}{
		{
			name: "success login record",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "My Login",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "user", Password: "pass"},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name: "success text record",
			body: CreateRecordRequest{
				Type:       "text",
				Name:       "My Note",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Text:       &TextPayload{Content: "some text"},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name: "success card record",
			body: CreateRecordRequest{
				Type:       "card",
				Name:       "My Card",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Card:       &CardPayload{Number: "4111111111111111", HolderName: "Test", ExpiryDate: "12/30", CVV: "123"},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name: "success binary record",
			body: CreateRecordRequest{
				Type:           "binary",
				Name:           "My File",
				DeviceID:       "device-1",
				KeyVersion:     1,
				PayloadVersion: 1,
				Binary:         &BinaryPayload{Data: []byte("data")},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name:       "invalid JSON body",
			body:       "not-json{{{",
			userID:     1,
			wantCode:   http.StatusBadRequest,
			serviceErr: nil,
		},
		{
			name: "missing record type",
			body: CreateRecordRequest{
				Name:       "Test",
				DeviceID:   "device-1",
				KeyVersion: 1,
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing name",
			body: CreateRecordRequest{
				Type:       "login",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing device_id",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "Test",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing key_version",
			body: CreateRecordRequest{
				Type:     "login",
				Name:     "Test",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing payload for login type",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "Test",
				DeviceID:   "device-1",
				KeyVersion: 1,
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "unauthorized - no user",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "Test",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   0,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "revision conflict",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "Test",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "u", Password: "p"},
			},
			userID:     1,
			serviceErr: models.ErrRevisionConflict,
			wantCode:   http.StatusConflict,
		},
		{
			name: "internal error",
			body: CreateRecordRequest{
				Type:       "login",
				Name:       "Test",
				DeviceID:   "device-1",
				KeyVersion: 1,
				Login:      &LoginPayload{Login: "u", Password: "p"},
			},
			userID:     1,
			serviceErr: errors.New("db error"),
			wantCode:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			mock := &recordServiceMock{err: tt.serviceErr}
			handler := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/records", bytes.NewReader(bodyBytes))
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

// recordServiceMock — простой мок для RecordService.
type recordServiceMock struct {
	err error
}

func (m *recordServiceMock) CreateRecord(record *models.Record) error {
	return m.err
}
