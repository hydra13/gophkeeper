//go:generate minimock -i .UserService -o mocks -s _mock.go -g
package auth_register_v1_post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// UserService описывает регистрацию пользователя.
type UserService interface {
	Register(ctx context.Context, email, password string) (int64, error)
}

// RegisterRequest описывает запрос на регистрацию.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse описывает ответ на регистрацию.
type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}

// Handler обрабатывает регистрацию пользователя.
type Handler struct {
	userService UserService
	log         zerolog.Logger
}

// NewHandler создаёт обработчик регистрации.
func NewHandler(userService UserService, log zerolog.Logger) *Handler {
	return &Handler{
		userService: userService,
		log:         log,
	}
}

// Handle регистрирует пользователя.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Debug().Err(err).Msg("register: failed to decode request body")
		responses.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateRegister(req); err != nil {
		h.log.Debug().Err(err).Msg("register: validation failed")
		responses.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, err := h.userService.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		mapError(w, h.log, err, "register")
		return
	}

	responses.JSON(w, http.StatusCreated, RegisterResponse{UserID: userID})
}

func validateRegister(req RegisterRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func mapError(w http.ResponseWriter, log zerolog.Logger, err error, op string) {
	switch {
	case errors.Is(err, models.ErrEmailAlreadyExists):
		responses.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, models.ErrInvalidCredentials):
		responses.Error(w, http.StatusUnauthorized, err.Error())
	default:
		log.Error().Err(err).Str("op", op).Msg("internal error")
		responses.Error(w, http.StatusInternalServerError, "internal error")
	}
}
