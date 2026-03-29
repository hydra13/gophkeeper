package sync_pull_v1_post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hydra13/gophkeeper/internal/models"
)

type mockSyncPuller struct {
	pullFunc func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error)
}

func (m *mockSyncPuller) Pull(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
	return m.pullFunc(userID, deviceID, sinceRevision, limit)
}

func TestSyncPullHandler_Success(t *testing.T) {
	mock := &mockSyncPuller{
		pullFunc: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			now := time.Now()
			deletedAt := now.Add(-time.Hour)
			return []models.RecordRevision{
					{ID: 1, RecordID: 1, UserID: 1, Revision: 10, DeviceID: "device-2"},
					{ID: 2, RecordID: 2, UserID: 1, Revision: 11, DeviceID: "device-3"},
				}, []models.Record{
					{
						ID:         1,
						UserID:     1,
						Type:       models.RecordTypeText,
						Name:       "first",
						Metadata:   "meta",
						Payload:    models.TextPayload{Content: "payload"},
						Revision:   10,
						DeletedAt:  &deletedAt,
						DeviceID:   "device-2",
						KeyVersion: 1,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
					{
						ID:         2,
						UserID:     1,
						Type:       models.RecordTypeText,
						Name:       "second",
						Payload:    models.TextPayload{Content: "payload2"},
						Revision:   11,
						DeviceID:   "device-3",
						KeyVersion: 1,
						CreatedAt:  now,
						UpdatedAt:  now,
					},
				}, []models.SyncConflict{
					{ID: 1, UserID: 1, RecordID: 2, LocalRevision: 9, ServerRevision: 11},
				}, nil
		},
	}

	h := NewHandler(mock)

	body, _ := json.Marshal(Request{
		UserID:        1,
		DeviceID:      "device-1",
		SinceRevision: 5,
		Limit:         50,
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
	if resp.NextRevision != 11 {
		t.Fatalf("expected next_revision 11, got %d", resp.NextRevision)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(resp.Records))
	}
	if len(resp.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(resp.Conflicts))
	}
	if !resp.Changes[0].Deleted {
		t.Fatal("expected deleted flag for first record to be true")
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
		pullFunc: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			capturedLimit = limit
			return nil, nil, nil, nil
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
		pullFunc: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			return nil, nil, nil, errors.New("internal error")
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
		pullFunc: func(userID int64, deviceID string, sinceRevision int64, limit int64) ([]models.RecordRevision, []models.Record, []models.SyncConflict, error) {
			// Возвращаем ровно limit записей → HasMore=true
			revs := make([]models.RecordRevision, limit)
			for i := range revs {
				revs[i] = models.RecordRevision{RecordID: int64(i) + 1, Revision: int64(i) + 1}
			}
			return revs, nil, nil, nil
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
