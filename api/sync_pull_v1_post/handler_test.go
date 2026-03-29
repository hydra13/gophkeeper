package sync_pull_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
)

type mockSyncPuller struct {
	pullFunc func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error)
}

func (m *mockSyncPuller) Pull(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
	return m.pullFunc(userID, deviceID, cursor, limit)
}

func TestSyncPullHandler_Success(t *testing.T) {
	mock := &mockSyncPuller{
		pullFunc: func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
			return []models.RecordRevision{
				{RecordID: 1, Revision: 10},
				{RecordID: 2, Revision: 11},
			}, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:   1,
		DeviceID: "device-1",
		Cursor:   5,
		Limit:    50,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(resp.Changes))
	}
	if resp.NextCursor != 11 {
		t.Fatalf("expected next_cursor 11, got %d", resp.NextCursor)
	}
}

func TestSyncPullHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockSyncPuller{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/pull", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestSyncPullHandler_InvalidBody(t *testing.T) {
	mock := &mockSyncPuller{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPullHandler_InvalidUserID(t *testing.T) {
	mock := &mockSyncPuller{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 0, DeviceID: "device-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPullHandler_EmptyDeviceID(t *testing.T) {
	mock := &mockSyncPuller{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPullHandler_DefaultLimit(t *testing.T) {
	var capturedLimit int64
	mock := &mockSyncPuller{
		pullFunc: func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
			capturedLimit = limit
			return nil, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: "device-1", Limit: 0})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if capturedLimit != 50 {
		t.Fatalf("expected default limit 50, got %d", capturedLimit)
	}
}

func TestSyncPullHandler_ServiceError(t *testing.T) {
	mock := &mockSyncPuller{
		pullFunc: func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
			return nil, errors.New("internal error")
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: "device-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestSyncPullHandler_HasMore(t *testing.T) {
	mock := &mockSyncPuller{
		pullFunc: func(userID int64, deviceID string, cursor int64, limit int64) ([]models.RecordRevision, error) {
			// Возвращаем ровно limit записей → HasMore=true
			revs := make([]models.RecordRevision, limit)
			for i := range revs {
				revs[i] = models.RecordRevision{RecordID: int64(i) + 1, Revision: int64(i) + 1}
			}
			return revs, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: "device-1", Limit: 3})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/pull", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.HasMore {
		t.Fatal("expected HasMore=true when results equal limit")
	}
}
