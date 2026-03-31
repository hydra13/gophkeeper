// Package auth_logout_v1_post реализует HTTP-ручку завершения сессии.
//
// POST /api/v1/auth/logout
//
//go:generate minimock -i .TokenService -o mocks -s _mock.go -g
package auth_logout_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/models"
)

// TokenService определяет зависимости, необходимые для завершения сессии.
type TokenService interface {
	Logout(ctx context.Context, accessToken string) error
}

// LogoutRequest — DTO запроса на завершение сессии.
type LogoutRequest struct{}

// LogoutResponse — DTO успешного ответа завершения сессии.
type LogoutResponse struct{}

// Handler обрабатывает запросы завершения сессии.
type Handler struct {
	tokenService TokenService
	log          zerolog.Logger
}

// NewHandler создаёт новый Handler для завершения сессии.
func NewHandler(tokenService TokenService, log zerolog.Logger) *Handler {
	return &Handler{
		tokenService: tokenService,
		log:          log,
	}
}

// Handle завершает текущую сессию по access token из заголовка Authorization.
// Требует заголовок Authorization: Bearer <access_token>.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		h.log.Debug().Msg("logout: missing authorization header")
		writeError(w, http.StatusUnauthorized, "authorization header required")
		return
	}

	err := h.tokenService.Logout(r.Context(), token)
	if err != nil {
		mapError(w, h.log, err, "logout")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil && status != http.StatusNoContent {
		json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrSessionExpired):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
