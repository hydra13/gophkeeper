package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowWithinLimit(t *testing.T) {
	limiter := NewRateLimiter(3, time.Second)

	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
}

func TestRateLimiter_DenyOverLimit(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	assert.True(t, limiter.Allow())
	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())
	assert.False(t, limiter.Allow())
}

func TestRateLimiter_WindowReset(t *testing.T) {
	limiter := NewRateLimiter(1, 50*time.Millisecond)

	assert.True(t, limiter.Allow())
	assert.False(t, limiter.Allow())

	// Wait for the window to expire.
	time.Sleep(80 * time.Millisecond)

	assert.True(t, limiter.Allow())
}

func TestRateLimitMiddleware_Allowed(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("allowed"))
	})

	middleware := RateLimit(limiter)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "allowed", rec.Body.String())
}

func TestRateLimitMiddleware_Denied(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimit(limiter)(handler)

	// First request passes.
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Second request is denied.
	req2 := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec2 := httptest.NewRecorder()
	middleware.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
}

func TestRateLimitMiddleware_MultipleDenials(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimit(limiter)(handler)

	// Exhaust the limit.
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Subsequent requests should all be 429.
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, r)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "request %d should be denied", i+1)
	}
}
