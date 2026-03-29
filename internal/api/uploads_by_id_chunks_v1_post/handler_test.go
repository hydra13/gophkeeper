package uploads_by_id_chunks_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockChunkUploader struct {
	uploadChunkFunc func(uploadID, chunkIndex int64, data []byte) (receivedChunks, totalChunks int64, completed bool, missingChunks []int64, err error)
}

func (m *mockChunkUploader) UploadChunk(uploadID, chunkIndex int64, data []byte) (receivedChunks, totalChunks int64, completed bool, missingChunks []int64, err error) {
	return m.uploadChunkFunc(uploadID, chunkIndex, data)
}

func TestChunkUploadHandler_Success(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 1, 3, false, []int64{1, 2}, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{
		ChunkIndex: 0,
		Data:       []byte("chunk-data"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/42/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp ChunkResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UploadID != 42 {
		t.Fatalf("expected upload_id 42, got %d", resp.UploadID)
	}
	if resp.ReceivedChunks != 1 {
		t.Fatalf("expected 1 received chunk, got %d", resp.ReceivedChunks)
	}
	if resp.Completed {
		t.Fatal("expected not completed")
	}
}

func TestChunkUploadHandler_Completed(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 3, 3, true, nil, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{
		ChunkIndex: 2,
		Data:       []byte("last-chunk"),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	var resp ChunkResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Completed {
		t.Fatal("expected completed")
	}
}

func TestChunkUploadHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockChunkUploader{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1/chunks", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestChunkUploadHandler_InvalidUploadID(t *testing.T) {
	mock := &mockChunkUploader{}
	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/invalid/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkUploadHandler_InvalidBody(t *testing.T) {
	mock := &mockChunkUploader{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkUploadHandler_NegativeChunkIndex(t *testing.T) {
	mock := &mockChunkUploader{}
	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: -1, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for negative chunk_index, got %d", w.Code)
	}
}

func TestChunkUploadHandler_EmptyData(t *testing.T) {
	mock := &mockChunkUploader{}
	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte{}})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty data, got %d", w.Code)
	}
}

func TestChunkUploadHandler_UploadNotFound(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, errors.New("upload session not found")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/99/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestChunkUploadHandler_UploadCompleted(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, errors.New("upload session already completed")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestChunkUploadHandler_UploadAborted(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, errors.New("upload session is aborted")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Fatalf("expected status 410, got %d", w.Code)
	}
}

func TestChunkUploadHandler_ChunkOutOfRange(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, errors.New("chunk index out of range")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 5, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestChunkUploadHandler_DuplicateChunk(t *testing.T) {
	mock := &mockChunkUploader{
		uploadChunkFunc: func(uploadID, chunkIndex int64, data []byte) (int64, int64, bool, []int64, error) {
			return 0, 0, false, nil, errors.New("chunk already received")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(ChunkRequest{ChunkIndex: 0, Data: []byte("data")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1/chunks", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}
}

func TestExtractUploadID(t *testing.T) {
	tests := []struct {
		path    string
		want    int64
		wantErr bool
	}{
		{"/api/v1/uploads/42/chunks", 42, false},
		{"/api/v1/uploads/1/chunks", 1, false},
		{"/api/v1/uploads/0/chunks", 0, false},
		{"/api/v1/uploads/abc/chunks", 0, true},
		{"/api/v1/uploads/chunks", 0, true},
	}
	for _, tt := range tests {
		got, err := extractUploadID(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractUploadID(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("extractUploadID(%q) = %d, want %d", tt.path, got, tt.want)
		}
	}
}
