//go:generate minimock -i .TokenService -o mocks -s _mock.go -g
package auth_logout_v1_post

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// TokenService описывает выход пользователя.
type TokenService interface {
	Logout(ctx context.Context, accessToken string) error
}

// LogoutRequest описывает запрос на выход.
type LogoutRequest struct{}

// LogoutResponse описывает ответ на выход.
type LogoutResponse struct{}

// Handler обрабатывает выход пользователя.
type Handler struct {
	tokenService TokenService
	log          zerolog.Logger
}

// NewHandler создаёт обработчик выхода.
func NewHandler(tokenService TokenService, log zerolog.Logger) *Handler {
	return &Handler{
		tokenService: tokenService,
		log:          log,
	}
}

// Handle выполняет выход пользователя.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		h.log.Debug().Msg("logout: missing authorization header")
		responses.Error(w, http.StatusUnauthorized, "authorization header required")
		return
	}

	err := h.tokenService.Logout(r.Context(), token)
	if err != nil {
		mapError(w, h.log, err, "logout")
		return
	}

	responses.NoContent(w)
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrUnauthorized):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrSessionExpired):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		responses.Error(w, http.StatusInternalServerError, "internal error")
	}
}
