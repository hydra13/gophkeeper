//go:generate minimock -i .HealthChecker -o mocks -s _mock.go -g
package health_v1_get

import (
	"encoding/json"
	"log"
	"net/http"
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := "ok"
	if err := h.checker.Health(); err != nil {
		status = "error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(Response{Status: status}); err != nil {
		log.Printf("health response encode failed: %v", err)
	}
}
