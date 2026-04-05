package middlewares

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter создаёт ограничитель запросов.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 || window <= 0 {
		return &RateLimiter{limiter: rate.NewLimiter(rate.Limit(0), 0)}
	}

	refillInterval := window / time.Duration(limit)
	if refillInterval <= 0 {
		refillInterval = time.Nanosecond
	}

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Every(refillInterval), limit),
	}
}

// Allow сообщает, можно ли пропустить запрос.
func (l *RateLimiter) Allow() bool {
	if l == nil || l.limiter == nil {
		return false
	}
	return l.limiter.Allow()
}

// RateLimit ограничивает частоту HTTP-запросов.
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
