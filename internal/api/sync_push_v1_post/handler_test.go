package sync_push_v1_post

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hydra13/gophkeeper/internal/models"
)

type mockSyncPusher struct {
	pushFunc func(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

func (m *mockSyncPusher) Push(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	return m.pushFunc(userID, changes)
}

func TestSyncPushHandler_Success(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return []models.RecordRevision{}, nil, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID: 1,
		Changes: []PendingChange{
			{RecordID: 1, Revision: 1, Operation: "update", DeviceID: "device-1"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Accepted != 1 {
		t.Fatalf("expected 1 accepted, got %d", resp.Accepted)
	}
	if len(resp.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(resp.Conflicts))
	}
}

func TestSyncPushHandler_WithConflicts(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return nil, []models.SyncConflict{
				{RecordID: 1, LocalRevision: 3, ServerRevision: 5},
			}, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID: 1,
		Changes: []PendingChange{
			{RecordID: 1, Revision: 3, Operation: "update", DeviceID: "device-1"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Accepted != 0 {
		t.Fatalf("expected 0 accepted, got %d", resp.Accepted)
	}
	if len(resp.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(resp.Conflicts))
	}
	if resp.Conflicts[0].ServerRevision != 5 {
		t.Fatalf("expected server_revision 5, got %d", resp.Conflicts[0].ServerRevision)
	}
}

func TestSyncPushHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockSyncPusher{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/push", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}

func TestSyncPushHandler_InvalidBody(t *testing.T) {
	mock := &mockSyncPusher{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPushHandler_InvalidUserID(t *testing.T) {
	mock := &mockSyncPusher{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 0, Changes: []PendingChange{{RecordID: 1}}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPushHandler_EmptyChanges(t *testing.T) {
	mock := &mockSyncPusher{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, Changes: []PendingChange{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPushHandler_ServiceError(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, changes []PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return nil, nil, models.ErrRevisionConflict
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID: 1,
		Changes: []PendingChange{{RecordID: 1, DeviceID: "d"}},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
