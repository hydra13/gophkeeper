package middlewares

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLS_RejectsNonTLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TLS()(handler)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	// By default, httptest.NewRequest creates a request with r.TLS == nil.
	assert.Nil(t, req.TLS)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "TLS required")
}

func TestTLS_AllowsTLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secure content"))
	})

	middleware := TLS()(handler)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	// Simulate a TLS connection by setting the TLS field.
	req.TLS = &tls.ConnectionState{}

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "secure content", rec.Body.String())
}

func TestTLS_RespondsWithCorrectErrorMessage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := TLS()(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/secret", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "TLS required\n", rec.Body.String())
}

func TestTLS_NextHandlerNotCalledWithoutTLS(t *testing.T) {
	nextCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := TLS()(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.False(t, nextCalled, "next handler should not be called when TLS is nil")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTLS_NextHandlerCalledWithTLS(t *testing.T) {
	nextCalled := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := TLS()(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	assert.True(t, nextCalled, "next handler should be called when TLS is present")
	assert.Equal(t, http.StatusOK, rec.Code)
}
