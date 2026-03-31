package records_by_id_binary_v1_get

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/records_by_id_binary_v1_get/mocks"
	"github.com/hydra13/gophkeeper/internal/middlewares"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		recordID  string
		userID    int64
		setupMock func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock)
		wantCode  int
	}{
		{
			name:     "unauthorized",
			recordID: "1",
			userID:   0,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc), mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "invalid record id",
			recordID: "abc",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc), mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "record not found",
			recordID: "42",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(42)).Return(nil, models.ErrRecordNotFound),
					mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "access denied",
			recordID: "1",
			userID:   2,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(&models.Record{
						ID:     1,
						UserID: 1,
						Type:   models.RecordTypeBinary,
						Name:   "secret.bin",
					}, nil),
					mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "record is not binary",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(&models.Record{
						ID:     1,
						UserID: 1,
						Type:   models.RecordTypeText,
						Name:   "note",
					}, nil),
					mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "create download session error",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				record := &models.Record{ID: 1, UserID: 1, Type: models.RecordTypeBinary, Name: "secret.bin"}
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(record, nil),
					mocks.NewUploadServiceMock(mc).
						CreateDownloadSessionMock.Expect(int64(1), int64(1)).Return(nil, errors.New("storage unavailable"))
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "download chunk error",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				record := &models.Record{ID: 1, UserID: 1, Type: models.RecordTypeBinary, Name: "secret.bin"}
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(record, nil),
					mocks.NewUploadServiceMock(mc).
						CreateDownloadSessionMock.Expect(int64(1), int64(1)).Return(&models.DownloadSession{
						ID:          10,
						RecordID:    1,
						UserID:      1,
						Status:      models.DownloadStatusActive,
						TotalChunks: 1,
					}, nil).
						DownloadChunkByIDMock.Expect(int64(10), int64(0)).Return(nil, errors.New("chunk backend error"))
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "confirm chunk error",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				record := &models.Record{ID: 1, UserID: 1, Type: models.RecordTypeBinary, Name: "secret.bin"}
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(record, nil),
					mocks.NewUploadServiceMock(mc).
						CreateDownloadSessionMock.Expect(int64(1), int64(1)).Return(&models.DownloadSession{
						ID:          10,
						RecordID:    1,
						UserID:      1,
						Status:      models.DownloadStatusActive,
						TotalChunks: 1,
					}, nil).
						DownloadChunkByIDMock.Expect(int64(10), int64(0)).Return(&models.Chunk{
						UploadID:   10,
						ChunkIndex: 0,
						Data:       []byte("chunk"),
					}, nil).
						ConfirmChunkMock.Expect(int64(10), int64(0)).Return(0, 0, "", errors.New("confirm failed"))
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "internal error",
			recordID: "1",
			userID:   1,
			setupMock: func(mc *minimock.Controller) (*mocks.RecordServiceMock, *mocks.UploadServiceMock) {
				return mocks.NewRecordServiceMock(mc).
						GetRecordMock.Expect(int64(1)).Return(nil, errors.New("db error")),
					mocks.NewUploadServiceMock(mc)
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mc := minimock.NewController(t)
			recordsMock, uploadsMock := tt.setupMock(mc)
			handler := NewHandler(recordsMock, uploadsMock)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/records/{id}/binary", nil)
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

func TestHandler_Handle_Success(t *testing.T) {
	t.Parallel()

	mc := minimock.NewController(t)
	record := &models.Record{
		ID:     7,
		UserID: 1,
		Type:   models.RecordTypeBinary,
		Name:   "archive.bin",
	}

	recordsMock := mocks.NewRecordServiceMock(mc).
		GetRecordMock.Expect(int64(7)).Return(record, nil)
	uploadsMock := mocks.NewUploadServiceMock(mc)
	uploadsMock.CreateDownloadSessionMock.Expect(int64(1), int64(7)).Return(&models.DownloadSession{
		ID:          99,
		RecordID:    7,
		UserID:      1,
		Status:      models.DownloadStatusActive,
		TotalChunks: 2,
	}, nil)
	uploadsMock.DownloadChunkByIDMock.When(int64(99), int64(0)).Then(&models.Chunk{
		UploadID:   99,
		ChunkIndex: 0,
		Data:       []byte("hello "),
	}, nil)
	uploadsMock.DownloadChunkByIDMock.When(int64(99), int64(1)).Then(&models.Chunk{
		UploadID:   99,
		ChunkIndex: 1,
		Data:       []byte("world"),
	}, nil)
	uploadsMock.ConfirmChunkMock.When(int64(99), int64(0)).Then(1, 2, models.DownloadStatusActive, nil)
	uploadsMock.ConfirmChunkMock.When(int64(99), int64(1)).Then(2, 2, models.DownloadStatusCompleted, nil)

	handler := NewHandler(recordsMock, uploadsMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records/{id}/binary", nil)
	req.SetPathValue("id", "7")
	req = req.WithContext(middlewares.ContextWithUserID(req.Context(), 1))

	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "hello world", rec.Body.String())
	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="archive.bin"`, rec.Header().Get("Content-Disposition"))
}

func TestHandler_Handle_DefaultFilename(t *testing.T) {
	t.Parallel()

	mc := minimock.NewController(t)
	recordsMock := mocks.NewRecordServiceMock(mc).
		GetRecordMock.Expect(int64(15)).Return(&models.Record{
		ID:     15,
		UserID: 1,
		Type:   models.RecordTypeBinary,
	}, nil)
	uploadsMock := mocks.NewUploadServiceMock(mc).
		CreateDownloadSessionMock.Expect(int64(1), int64(15)).Return(&models.DownloadSession{
		ID:          5,
		RecordID:    15,
		UserID:      1,
		Status:      models.DownloadStatusActive,
		TotalChunks: 1,
	}, nil).
		DownloadChunkByIDMock.Expect(int64(5), int64(0)).Return(&models.Chunk{
		UploadID:   5,
		ChunkIndex: 0,
		Data:       []byte("bin"),
	}, nil).
		ConfirmChunkMock.Expect(int64(5), int64(0)).Return(1, 1, models.DownloadStatusCompleted, nil)

	handler := NewHandler(recordsMock, uploadsMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/records/{id}/binary", nil)
	req.SetPathValue("id", "15")
	req = req.WithContext(middlewares.ContextWithUserID(req.Context(), 1))

	rec := httptest.NewRecorder()
	handler.Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, `attachment; filename="record-15.bin"`, rec.Header().Get("Content-Disposition"))
}
