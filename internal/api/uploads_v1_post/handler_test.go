package uploads_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockUploadCreator struct {
	createSessionFunc func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error)
}

func (m *mockUploadCreator) CreateSession(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
	return m.createSessionFunc(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion)
}

func TestUploadsCreateHandler_Success(t *testing.T) {
	mock := &mockUploadCreator{
		createSessionFunc: func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
			return 42, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:      1,
		RecordID:    10,
		TotalChunks: 4,
		ChunkSize:   1024,
		TotalSize:   4096,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UploadID != 42 {
		t.Fatalf("expected upload_id 42, got %d", resp.UploadID)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", resp.Status)
	}
}

func TestUploadsCreateHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidBody(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidUserID(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 0, RecordID: 1, TotalChunks: 2, ChunkSize: 1024, TotalSize: 2048})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidRecordID(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, RecordID: 0, TotalChunks: 2, ChunkSize: 1024, TotalSize: 2048})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid record_id, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidTotalChunks(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, RecordID: 1, TotalChunks: 0, ChunkSize: 1024, TotalSize: 2048})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid total_chunks, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidChunkSize(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, RecordID: 1, TotalChunks: 2, ChunkSize: 0, TotalSize: 2048})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid chunk_size, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_InvalidTotalSize(t *testing.T) {
	mock := &mockUploadCreator{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, RecordID: 1, TotalChunks: 2, ChunkSize: 1024, TotalSize: 0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid total_size, got %d", w.Code)
	}
}

func TestUploadsCreateHandler_ServiceError(t *testing.T) {
	mock := &mockUploadCreator{
		createSessionFunc: func(userID, recordID, totalChunks, chunkSize, totalSize, keyVersion int64) (int64, error) {
			return 0, errors.New("db unavailable")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, RecordID: 1, TotalChunks: 2, ChunkSize: 1024, TotalSize: 2048})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
