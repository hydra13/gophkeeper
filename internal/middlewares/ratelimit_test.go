package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
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
	synctest.Test(t, func(t *testing.T) {
		limiter := newRateLimiterWithClock(1, 50*time.Millisecond, time.Now)

		assert.True(t, limiter.Allow())
		assert.False(t, limiter.Allow())

		time.Sleep(50 * time.Millisecond)

		assert.True(t, limiter.Allow())
	})
}

func TestNewRateLimiterWithClock_NilClockUsesDefault(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiterWithClock(1, time.Second, nil)

	assert.NotNil(t, limiter.now)
	assert.NotNil(t, limiter.limiter)
}

func TestNewRateLimiterWithClock_InvalidInputDisablesLimiter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		limit  int
		window time.Duration
	}{
		{
			name:   "non-positive limit",
			limit:  0,
			window: time.Second,
		},
		{
			name:   "non-positive window",
			limit:  1,
			window: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			limiter := newRateLimiterWithClock(tc.limit, tc.window, time.Now)
			assert.False(t, limiter.Allow())
		})
	}
}

func TestNewRateLimiterWithClock_ClampsTinyRefillInterval(t *testing.T) {
	t.Parallel()

	limiter := newRateLimiterWithClock(10, time.Nanosecond, time.Now)

	assert.Equal(t, rate.Every(time.Nanosecond), limiter.limiter.Limit())
}

func TestRateLimiterAllow_NilLimiter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		limiter *RateLimiter
	}{
		{
			name: "nil receiver",
		},
		{
			name:    "nil inner limiter",
			limiter: &RateLimiter{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, tc.limiter.Allow())
		})
	}
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
