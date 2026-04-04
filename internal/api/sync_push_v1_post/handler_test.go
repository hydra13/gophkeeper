package sync_push_v1_post

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	recordscommon "github.com/hydra13/gophkeeper/internal/api/records_common"
	"github.com/hydra13/gophkeeper/internal/models"
	"github.com/stretchr/testify/require"
)

type mockSyncPusher struct {
	pushFunc func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error)
}

func (m *mockSyncPusher) Push(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
	return m.pushFunc(userID, deviceID, changes)
}

func TestSyncPushHandler_Success(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return []models.RecordRevision{
				{ID: 1, RecordID: 1, UserID: userID, Revision: 2, DeviceID: deviceID},
			}, nil, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:   1,
		DeviceID: "device-1",
		Changes: []PendingChange{
			{
				Record: recordscommon.RecordDTO{
					ID:         1,
					UserID:     1,
					Type:       "text",
					Name:       "name",
					Revision:   1,
					DeviceID:   "device-1",
					KeyVersion: 1,
					CreatedAt:  "2024-01-01T00:00:00Z",
					UpdatedAt:  "2024-01-01T00:00:00Z",
					Payload:    recordscommon.TextPayloadDTO{Content: "payload"},
				},
				BaseRevision: 1,
			},
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
	if len(resp.Accepted) != 1 {
		t.Fatalf("expected 1 accepted, got %d", len(resp.Accepted))
	}
	if len(resp.Conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(resp.Conflicts))
	}
}

func TestSyncPushHandler_WithConflicts(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return nil, []models.SyncConflict{
				{RecordID: 1, LocalRevision: 3, ServerRevision: 5},
			}, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:   1,
		DeviceID: "device-1",
		Changes: []PendingChange{
			{
				Record: recordscommon.RecordDTO{
					ID:         1,
					UserID:     1,
					Type:       "text",
					Name:       "name",
					Revision:   3,
					DeviceID:   "device-1",
					KeyVersion: 1,
					CreatedAt:  "2024-01-01T00:00:00Z",
					UpdatedAt:  "2024-01-01T00:00:00Z",
					Payload:    recordscommon.TextPayloadDTO{Content: "payload"},
				},
				BaseRevision: 3,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
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

	body, _ := json.Marshal(Request{UserID: 0, DeviceID: "device-1", Changes: []PendingChange{{}}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPushHandler_EmptyDeviceID(t *testing.T) {
	mock := &mockSyncPusher{}
	h := NewHandler(mock)

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: "", Changes: []PendingChange{{}}})
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

	body, _ := json.Marshal(Request{UserID: 1, DeviceID: "device-1", Changes: []PendingChange{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestSyncPushHandler_ServiceError(t *testing.T) {
	mock := &mockSyncPusher{
		pushFunc: func(userID int64, deviceID string, changes []models.PendingChange) ([]models.RecordRevision, []models.SyncConflict, error) {
			return nil, nil, models.ErrRevisionConflict
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:   1,
		DeviceID: "d",
		Changes: []PendingChange{
			{
				Record: recordscommon.RecordDTO{
					ID:         1,
					UserID:     1,
					Type:       "text",
					Name:       "name",
					Revision:   1,
					DeviceID:   "d",
					KeyVersion: 1,
					CreatedAt:  "2024-01-01T00:00:00Z",
					UpdatedAt:  "2024-01-01T00:00:00Z",
					Payload:    recordscommon.TextPayloadDTO{Content: "payload"},
				},
				BaseRevision: 1,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
