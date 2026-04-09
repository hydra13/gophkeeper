package uploads_by_id_v1_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/uploads_by_id_v1_get/mocks"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		path      string
		setupMock func(mc *minimock.Controller) UploadStatusGetter
		wantCode  int
		wantResp  *models.UploadStatusResponse
	}{
		{
			name:   "success",
			method: http.MethodGet,
			path:   "/api/v1/uploads/1",
			setupMock: func(mc *minimock.Controller) UploadStatusGetter {
				return mocks.NewUploadStatusGetterMock(mc).
					GetUploadStatusMock.
					Expect(int64(1)).
					Return(
						&models.UploadStatusResponse{
							UploadID:       1,
							RecordID:       10,
							Status:         "pending",
							TotalChunks:    4,
							ReceivedChunks: 2,
							MissingChunks:  []int64{2, 3},
						}, nil)
			},
			wantCode: http.StatusOK,
			wantResp: &models.UploadStatusResponse{
				UploadID:       1,
				RecordID:       10,
				Status:         "pending",
				TotalChunks:    4,
				ReceivedChunks: 2,
				MissingChunks:  []int64{2, 3},
			},
		},
		{
			name:   "method not allowed",
			method: http.MethodPost,
			path:   "/api/v1/uploads/1",
			setupMock: func(mc *minimock.Controller) UploadStatusGetter {
				return mocks.NewUploadStatusGetterMock(mc)
			},
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid upload id",
			method: http.MethodGet,
			path:   "/api/v1/uploads/invalid",
			setupMock: func(mc *minimock.Controller) UploadStatusGetter {
				return mocks.NewUploadStatusGetterMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/api/v1/uploads/99",
			setupMock: func(mc *minimock.Controller) UploadStatusGetter {
				return mocks.NewUploadStatusGetterMock(mc).
					GetUploadStatusMock.Expect(int64(99)).Return(nil, errors.New("upload session not found"))
			},
			wantCode: http.StatusNotFound,
		},
		{
			name:   "service error",
			method: http.MethodGet,
			path:   "/api/v1/uploads/1",
			setupMock: func(mc *minimock.Controller) UploadStatusGetter {
				return mocks.NewUploadStatusGetterMock(mc).
					GetUploadStatusMock.Expect(int64(1)).Return(nil, errors.New("unexpected error"))
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			handler := NewHandler(tt.setupMock(mc))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code, rec.Body.String())
			if tt.wantResp == nil {
				return
			}

			var resp models.UploadStatusResponse
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
			assert.Equal(t, *tt.wantResp, resp)
		})
	}
}

func TestHandler_ExtractUploadID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		want    int64
		wantErr bool
	}{
		{path: "/api/v1/uploads/42", want: 42},
		{path: "/api/v1/uploads/1", want: 1},
		{path: "/api/v1/uploads/abc", wantErr: true},
		{path: "/api/v1/uploads", wantErr: true},
	}

	for _, tt := range tests {
		got, err := extractUploadID(tt.path)
		if tt.wantErr {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}
