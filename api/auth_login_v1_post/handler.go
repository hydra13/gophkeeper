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

// UserService определяет зависимости, необходимые для логина.
type UserService interface {
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
}

// LoginRequest — DTO запроса на логин.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse — DTO успешного ответа логина.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Handler обрабатывает HTTP-запросы на логин.
type Handler struct {
	userService UserService
	log         zerolog.Logger
}

// NewHandler создаёт новый Handler для логина.
func NewHandler(userService UserService, log zerolog.Logger) *Handler {
	return &Handler{
		userService: userService,
		log:         log,
	}
}

// Handle обрабатывает POST /api/v1/auth/login.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("login: failed to decode request body")
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateLogin(req); err != nil {
		h.log.Debug().Err(err).Msg("login: validation failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	accessToken, refreshToken, err := h.userService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		mapError(w, h.log, err, "login")
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
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
	return nil
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
	case errors.Is(err, models.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, models.ErrUserNotFound):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
