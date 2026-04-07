//go:generate minimock -i .TokenService -o mocks -s _mock.go -g
package auth_refresh_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// TokenService описывает обновление токенов.
type TokenService interface {
	Refresh(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error)
}

// RefreshRequest описывает запрос на обновление токенов.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse описывает ответ с новыми токенами.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Handler обрабатывает обновление токенов.
type Handler struct {
	tokenService TokenService
	log          zerolog.Logger
}

// NewHandler создаёт обработчик обновления токенов.
func NewHandler(tokenService TokenService, log zerolog.Logger) *Handler {
	return &Handler{
		tokenService: tokenService,
		log:          log,
	}
}

// Handle обновляет пару токенов.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("refresh: failed to decode request body")
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.log.Debug().Msg("refresh: refresh_token is required")
		responses.Error(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	accessToken, refreshToken, err := h.tokenService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		mapError(w, h.log, err, "refresh")
		return
	}

	responses.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrSessionExpired), errors.Is(err, models.ErrSessionRevoked):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrUnauthorized):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		responses.Error(w, http.StatusInternalServerError, "internal error")
	}
}
