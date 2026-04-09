package records_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/stretchr/testify/require"
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
				Type:     "login",
				Name:     "My Login",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "user", Password: "pass"},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name: "success text record",
			body: CreateRecordRequest{
				Type:     "text",
				Name:     "My Note",
				DeviceID: "device-1",
				Text:     &TextPayload{Content: "some text"},
			},
			userID:   1,
			wantCode: http.StatusCreated,
		},
		{
			name: "success card record",
			body: CreateRecordRequest{
				Type:     "card",
				Name:     "My Card",
				DeviceID: "device-1",
				Card:     &CardPayload{Number: "4111111111111111", HolderName: "Test", ExpiryDate: "12/30", CVV: "123"},
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
				PayloadVersion: 1,
				Binary:         &BinaryPayload{},
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
				Name:     "Test",
				DeviceID: "device-1",
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing name",
			body: CreateRecordRequest{
				Type:     "login",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing device_id",
			body: CreateRecordRequest{
				Type:  "login",
				Name:  "Test",
				Login: &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "missing payload for login type",
			body: CreateRecordRequest{
				Type:     "login",
				Name:     "Test",
				DeviceID: "device-1",
			},
			userID:   1,
			wantCode: http.StatusBadRequest,
		},
		{
			name: "unauthorized - no user",
			body: CreateRecordRequest{
				Type:     "login",
				Name:     "Test",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "u", Password: "p"},
			},
			userID:   0,
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "revision conflict",
			body: CreateRecordRequest{
				Type:     "login",
				Name:     "Test",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "u", Password: "p"},
			},
			userID:     1,
			serviceErr: models.ErrRevisionConflict,
			wantCode:   http.StatusConflict,
		},
		{
			name: "internal error",
			body: CreateRecordRequest{
				Type:     "login",
				Name:     "Test",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "u", Password: "p"},
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

func TestRequestToRecord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     CreateRecordRequest
		wantErr string
	}{
		{
			name: "login success",
			req: CreateRecordRequest{
				Type:     "login",
				Name:     "Name",
				DeviceID: "device-1",
				Login:    &LoginPayload{Login: "user", Password: "pass"},
			},
		},
		{
			name: "text success",
			req: CreateRecordRequest{
				Type:     "text",
				Name:     "Name",
				DeviceID: "device-1",
				Text:     &TextPayload{Content: "text"},
			},
		},
		{
			name: "binary success",
			req: CreateRecordRequest{
				Type:           "binary",
				Name:           "Name",
				DeviceID:       "device-1",
				PayloadVersion: 1,
				Binary:         &BinaryPayload{},
			},
		},
		{
			name: "card success",
			req: CreateRecordRequest{
				Type:     "card",
				Name:     "Name",
				DeviceID: "device-1",
				Card:     &CardPayload{Number: "4111111111111111", HolderName: "Test", ExpiryDate: "12/30", CVV: "123"},
			},
		},
		{
			name: "invalid type",
			req: CreateRecordRequest{
				Type:     "nope",
				Name:     "Name",
				DeviceID: "device-1",
			},
			wantErr: "invalid record type",
		},
		{
			name: "missing login payload",
			req: CreateRecordRequest{
				Type:     "login",
				Name:     "Name",
				DeviceID: "device-1",
			},
			wantErr: "login payload is required",
		},
		{
			name: "missing text payload",
			req: CreateRecordRequest{
				Type:     "text",
				Name:     "Name",
				DeviceID: "device-1",
			},
			wantErr: "text payload is required",
		},
		{
			name: "missing binary payload version",
			req: CreateRecordRequest{
				Type:     "binary",
				Name:     "Name",
				DeviceID: "device-1",
				Binary:   &BinaryPayload{},
			},
			wantErr: models.ErrInvalidPayloadVersion.Error(),
		},
		{
			name: "missing card payload",
			req: CreateRecordRequest{
				Type:     "card",
				Name:     "Name",
				DeviceID: "device-1",
			},
			wantErr: "card payload is required",
		},
		{
			name: "invalid card payload",
			req: CreateRecordRequest{
				Type:     "card",
				Name:     "Name",
				DeviceID: "device-1",
				Card:     &CardPayload{Number: "123", HolderName: "Test", ExpiryDate: "12/30", CVV: "123"},
			},
			wantErr: "invalid card number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record, err := requestToRecord(&tt.req)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Nil(t, record)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, record)
			require.Equal(t, tt.req.Name, record.Name)
			require.Equal(t, tt.req.DeviceID, record.DeviceID)
			require.Equal(t, models.RecordType(tt.req.Type), record.Type)
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
