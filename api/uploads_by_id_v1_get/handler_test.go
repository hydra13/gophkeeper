package uploads_by_id_v1_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockUploadStatusGetter struct {
	getUploadStatusFunc func(uploadID int64) (*UploadStatusResponse, error)
}

func (m *mockUploadStatusGetter) GetUploadStatus(uploadID int64) (*UploadStatusResponse, error) {
	return m.getUploadStatusFunc(uploadID)
}

func TestUploadStatusHandler_Success(t *testing.T) {
	mock := &mockUploadStatusGetter{
		getUploadStatusFunc: func(uploadID int64) (*UploadStatusResponse, error) {
			return &UploadStatusResponse{
				UploadID:       1,
				RecordID:       10,
				Status:         "pending",
				TotalChunks:    4,
				ReceivedChunks: 2,
				MissingChunks:  []int64{2, 3},
			}, nil
		},
	}

	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp UploadStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UploadID != 1 {
		t.Fatalf("expected upload_id 1, got %d", resp.UploadID)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", resp.Status)
	}
	if len(resp.MissingChunks) != 2 {
		t.Fatalf("expected 2 missing chunks, got %d", len(resp.MissingChunks))
	}
}

func TestUploadStatusHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockUploadStatusGetter{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestUploadStatusHandler_InvalidUploadID(t *testing.T) {
	mock := &mockUploadStatusGetter{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/invalid", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestUploadStatusHandler_NotFound(t *testing.T) {
	mock := &mockUploadStatusGetter{
		getUploadStatusFunc: func(uploadID int64) (*UploadStatusResponse, error) {
			return nil, errors.New("upload session not found")
		},
	}

	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/99", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestUploadStatusHandler_ServiceError(t *testing.T) {
	mock := &mockUploadStatusGetter{
		getUploadStatusFunc: func(uploadID int64) (*UploadStatusResponse, error) {
			return nil, errors.New("unexpected error")
		},
	}

	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestExtractUploadID(t *testing.T) {
	tests := []struct {
		path    string
		want    int64
		wantErr bool
	}{
		{"/api/v1/uploads/42", 42, false},
		{"/api/v1/uploads/1", 1, false},
		{"/api/v1/uploads/abc", 0, true},
		{"/api/v1/uploads", 0, true},
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
