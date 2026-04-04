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

type TokenService interface {
	Logout(ctx context.Context, accessToken string) error
}

type LogoutRequest struct{}

type LogoutResponse struct{}

type Handler struct {
	tokenService TokenService
	log          zerolog.Logger
}

func NewHandler(tokenService TokenService, log zerolog.Logger) *Handler {
	return &Handler{
		tokenService: tokenService,
		log:          log,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		h.log.Debug().Msg("logout: missing authorization header")
		writeError(h.log, w, http.StatusUnauthorized, "authorization header required")
		return
	}

	err := h.tokenService.Logout(r.Context(), token)
	if err != nil {
		mapError(w, h.log, err, "logout")
		return
	}

	writeJSON(h.log, w, http.StatusNoContent, nil)
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

func writeJSON(log zerolog.Logger, w http.ResponseWriter, status int, v interface{}) {
	if v != nil && status != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Error().Err(err).Msg("logout response encode failed")
		}
		return
	}
	w.WriteHeader(status)
}

func writeError(log zerolog.Logger, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Error().Err(err).Msg("logout error response encode failed")
	}
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrUnauthorized):
		writeError(log, w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrSessionExpired):
		writeError(log, w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		writeError(log, w, http.StatusInternalServerError, "internal error")
	}
}
