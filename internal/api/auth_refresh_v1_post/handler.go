// Package auth_refresh_v1_post реализует HTTP-ручку обновления токенов.
//
// POST /api/v1/auth/refresh
//
//go:generate minimock -i .TokenService -o mocks -s _mock.go -g
package auth_refresh_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/models"
)

// TokenService определяет зависимости, необходимые для обновления токена.
type TokenService interface {
	Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error)
}

// RefreshRequest — DTO запроса на обновление токена.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse — DTO успешного ответа обновления токена.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Handler обрабатывает запросы обновления токенов.
type Handler struct {
	tokenService TokenService
	log          zerolog.Logger
}

// NewHandler создаёт новый Handler для обновления токена.
func NewHandler(tokenService TokenService, log zerolog.Logger) *Handler {
	return &Handler{
		tokenService: tokenService,
		log:          log,
	}
}

// Handle обновляет access и refresh токены по refresh_token.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("refresh: failed to decode request body")
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.log.Debug().Msg("refresh: refresh_token is required")
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	accessToken, refreshToken, err := h.tokenService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		mapError(w, h.log, err, "refresh")
		return
	}

	writeJSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrSessionExpired), errors.Is(err, models.ErrSessionRevoked):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
