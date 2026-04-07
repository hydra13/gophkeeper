//go:generate minimock -i .HealthChecker -o mocks -s _mock.go -g
package health_v1_get

import (
	"net/http"

	"github.com/hydra13/gophkeeper/internal/api/responses"
)

// Response описывает ответ health-check.
type Response struct {
	Status string `json:"status"`
}

// HealthChecker проверяет состояние сервиса.
type HealthChecker interface {
	Health() error
}

// Handler обрабатывает health-check.
type Handler struct {
	checker HealthChecker
}

// NewHandler создаёт обработчик health-check.
func NewHandler(checker HealthChecker) *Handler {
	return &Handler{checker: checker}
}

// ServeHTTP возвращает состояние сервиса.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := "ok"
	if err := h.checker.Health(); err != nil {
		status = "error"
	}

	responses.JSON(w, http.StatusOK, Response{Status: status})
}
