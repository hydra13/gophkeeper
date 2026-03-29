package middlewares

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter — простой лимитер запросов на окно.
type RateLimiter struct {
	mu          sync.Mutex
	windowStart time.Time
	count       int
	limit       int
	window      time.Duration
}

// NewRateLimiter создаёт новый лимитер.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		windowStart: time.Now(),
		limit:       limit,
		window:      window,
	}
}

// Allow проверяет, можно ли пропустить запрос.
func (l *RateLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if time.Since(l.windowStart) > l.window {
		l.windowStart = time.Now()
		l.count = 0
	}
	if l.count >= l.limit {
		return false
	}
	l.count++
	return true
}

// RateLimit middleware для ограничения запросов.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
