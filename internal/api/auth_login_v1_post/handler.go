//go:generate minimock -i .UserService -o mocks -s _mock.go -g
package auth_login_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/models"
)

type UserService interface {
	Login(ctx context.Context, email, password, deviceID, deviceName, clientType string) (accessToken, refreshToken string, err error)
}

type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	ClientType string `json:"client_type"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Handler struct {
	userService UserService
	log         zerolog.Logger
}

func NewHandler(userService UserService, log zerolog.Logger) *Handler {
	return &Handler{
		userService: userService,
		log:         log,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("login: failed to decode request body")
		writeError(h.log, w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateLogin(req); err != nil {
		h.log.Debug().Err(err).Msg("login: validation failed")
		writeError(h.log, w, http.StatusBadRequest, err.Error())
		return
	}

	accessToken, refreshToken, err := h.userService.Login(r.Context(), req.Email, req.Password, req.DeviceID, req.DeviceName, req.ClientType)
	if err != nil {
		mapError(w, h.log, err, "login")
		return
	}

	writeJSON(h.log, w, http.StatusOK, LoginResponse{
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

func writeJSON(log zerolog.Logger, w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("login response encode failed")
	}
}

func writeError(log zerolog.Logger, w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Error().Err(err).Msg("login error response encode failed")
	}
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrInvalidCredentials):
		writeError(log, w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrUserNotFound):
		writeError(log, w, http.StatusUnauthorized, "invalid credentials")
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		writeError(log, w, http.StatusInternalServerError, "internal error")
	}
}
