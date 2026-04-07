//go:generate minimock -i .UserService -o mocks -s _mock.go -g
package auth_login_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// UserService описывает вход пользователя.
type UserService interface {
	Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (accessToken, refreshToken string, err error)
}

// LoginRequest описывает запрос на вход.
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	ClientType string `json:"client_type"`
}

// LoginResponse описывает ответ на вход.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Handler обрабатывает вход пользователя.
type Handler struct {
	userService UserService
	log         zerolog.Logger
}

// NewHandler создаёт обработчик входа.
func NewHandler(userService UserService, log zerolog.Logger) *Handler {
	return &Handler{
		userService: userService,
		log:         log,
	}
}

// Handle выполняет вход пользователя.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("login: failed to decode request body")
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateLogin(req); err != nil {
		h.log.Debug().Err(err).Msg("login: validation failed")
		responses.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	accessToken, refreshToken, err := h.userService.Login(r.Context(), req.Email, req.Password, req.DeviceID, req.DeviceName, req.ClientType)
	if err != nil {
		mapError(w, h.log, err, "login")
		return
	}

	responses.JSON(w, http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func validateLogin(req LoginRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if req.DeviceID == "" {
		return errors.New("device_id is required")
	}
	return nil
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrInvalidCredentials):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrUserNotFound):
		responses.Error(w, http.StatusUnauthorized, "invalid credentials")
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		responses.Error(w, http.StatusInternalServerError, "internal error")
	}
}
