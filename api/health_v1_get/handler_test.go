package health_v1_get

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockHealthChecker struct {
	healthFunc func() error
}

func (m *mockHealthChecker) Health() error {
	return m.healthFunc()
}

func TestHealthHandler_OK(t *testing.T) {
	mock := &mockHealthChecker{
		healthFunc: func() error { return nil },
	}

	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %s", resp.Status)
	}
}

func TestHealthHandler_Error(t *testing.T) {
	mock := &mockHealthChecker{
		healthFunc: func() error { return errors.New("db unavailable") },
	}

	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "error" {
		t.Fatalf("expected status 'error', got %s", resp.Status)
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	mock := &mockHealthChecker{}
	h := NewHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", w.Code)
	}
}
