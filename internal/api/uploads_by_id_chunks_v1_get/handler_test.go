package uploads_by_id_chunks_v1_get

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockChunkDownloader struct {
	downloadChunkFunc func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error)
}

func (m *mockChunkDownloader) DownloadChunk(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
	return m.downloadChunkFunc(uploadID, chunkIndex)
}

func TestChunkDownloadHandler_Success(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return &ChunkDownloadResponse{
				UploadID:        uploadID,
				DownloadID:      10,
				RecordID:        7,
				ChunkIndex:      chunkIndex,
				Data:            []byte("chunk-data"),
				TotalChunks:     3,
				ConfirmedChunks: 1,
				RemainingChunks: []int64{2},
				Completed:       false,
			}, nil
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/42/chunks/1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockChunkDownloader{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_InvalidPath(t *testing.T) {
	mock := &mockChunkDownloader{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_NegativeChunkIndex(t *testing.T) {
	mock := &mockChunkDownloader{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/-1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_NotFound(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("download session not found")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_Completed(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("download session already completed")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_Aborted(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("download session is aborted")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Fatalf("expected status 410, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_NotActive(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("download session is not active")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_UploadNotPending(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("upload session is not pending")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_ChunkOutOfRange(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("chunk index out of range")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/5", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_ChunkAlreadyConfirmed(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("chunk already confirmed by client")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/0", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestChunkDownloadHandler_ChunkOutOfOrder(t *testing.T) {
	mock := &mockChunkDownloader{
		downloadChunkFunc: func(uploadID, chunkIndex int64) (*ChunkDownloadResponse, error) {
			return nil, errors.New("chunk order violated")
		},
	}

	h := NewHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks/2", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestExtractUploadIDAndChunkIndex(t *testing.T) {
	tests := []struct {
		path    string
		wantID  int64
		wantIdx int64
		wantErr bool
	}{
		{"/api/v1/uploads/42/chunks/1", 42, 1, false},
		{"/api/v1/uploads/1/chunks/0", 1, 0, false},
		{"/api/v1/uploads/abc/chunks/0", 0, 0, true},
		{"/api/v1/uploads/1/chunks", 0, 0, true},
		{"/api/v1/uploads/chunks/1", 0, 0, true},
	}
	for _, tt := range tests {
		id, idx, err := extractUploadIDAndChunkIndex(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractUploadIDAndChunkIndex(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
		}
		if id != tt.wantID || idx != tt.wantIdx {
			t.Errorf("extractUploadIDAndChunkIndex(%q) = (%d, %d), want (%d, %d)", tt.path, id, idx, tt.wantID, tt.wantIdx)
		}
	}
}
