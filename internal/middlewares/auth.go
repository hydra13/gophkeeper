package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// TokenValidator проверяет токен сессии.
type TokenValidator interface {
	ValidateSession(token string) (int64, error)
}

type userIDContextKey struct{}

// UserIDFromContext возвращает идентификатор пользователя из контекста.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(int64)
	return userID, ok
}

// ContextWithUserID сохраняет идентификатор пользователя в контексте.
func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// Auth проверяет bearer-токен и добавляет user_id в контекст.
func Auth(validator TokenValidator, log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token == "" {
				http.Error(w, "empty token", http.StatusUnauthorized)
				return
			}
			userID, err := validator.ValidateSession(token)
			if err != nil {
				log.Debug().Err(err).Msg("auth session validation failed")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := ContextWithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
