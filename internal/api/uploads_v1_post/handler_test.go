package uploads_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/uploads_v1_post/mocks"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	validRequest := Request{
		UserID:      1,
		RecordID:    10,
		TotalChunks: 4,
		ChunkSize:   1024,
		TotalSize:   4096,
		KeyVersion:  7,
	}

	tests := []struct {
		name       string
		method     string
		body       interface{}
		setupMock  func(mc *minimock.Controller) UploadCreator
		wantCode   int
		wantUpload int64
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   validRequest,
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc).
					CreateSessionMock.Expect(int64(1), int64(10), int64(4), int64(1024), int64(4096), int64(7)).
					Return(int64(42), nil)
			},
			wantCode:   http.StatusCreated,
			wantUpload: 42,
		},
		{
			name:   "method not allowed",
			method: http.MethodGet,
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid body",
			method: http.MethodPost,
			body:   "invalid",
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid user id",
			method: http.MethodPost,
			body: Request{
				UserID:      0,
				RecordID:    10,
				TotalChunks: 4,
				ChunkSize:   1024,
				TotalSize:   4096,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid record id",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    0,
				TotalChunks: 4,
				ChunkSize:   1024,
				TotalSize:   4096,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid total chunks",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    10,
				TotalChunks: 0,
				ChunkSize:   1024,
				TotalSize:   4096,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid chunk size",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    10,
				TotalChunks: 4,
				ChunkSize:   0,
				TotalSize:   4096,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid total size",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    10,
				TotalChunks: 4,
				ChunkSize:   1024,
				TotalSize:   0,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "invalid key version",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    10,
				TotalChunks: 4,
				ChunkSize:   1024,
				TotalSize:   4096,
				KeyVersion:  0,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "service error",
			method: http.MethodPost,
			body: Request{
				UserID:      1,
				RecordID:    10,
				TotalChunks: 4,
				ChunkSize:   1024,
				TotalSize:   4096,
				KeyVersion:  7,
			},
			setupMock: func(mc *minimock.Controller) UploadCreator {
				return mocks.NewUploadCreatorMock(mc).
					CreateSessionMock.Expect(int64(1), int64(10), int64(4), int64(1024), int64(4096), int64(7)).
					Return(int64(0), errors.New("db unavailable"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			handler := NewHandler(tt.setupMock(mc))

			var bodyBytes []byte
			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/api/v1/uploads", bytes.NewReader(bodyBytes))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			if tt.wantCode != http.StatusCreated {
				return
			}

			var resp Response
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, tt.wantUpload, resp.UploadID)
			assert.Equal(t, "pending", resp.Status)
		})
	}
}
