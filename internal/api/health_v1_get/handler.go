// Package health_v1_get implements the HTTP endpoint for health checks.
//
// GET /api/v1/health
//
// Returns service health status. Used by load balancers and monitoring systems.
package health_v1_get

import (
	"encoding/json"
	"net/http"
)

// Response — DTO ответа health-check.
type Response struct {
	// Status — статус сервиса: "ok" или "error".
	Status string `json:"status"`
}

// HealthChecker — интерфейс проверки здоровья сервиса.
type HealthChecker interface {
	// Health возвращает nil если сервис здоров.
	Health() error
}

// Handler обрабатывает GET /api/v1/health.
type Handler struct {
	checker HealthChecker
}

// NewHandler создаёт новый обработчик health-check.
func NewHandler(checker HealthChecker) *Handler {
	return &Handler{checker: checker}
}

// ServeHTTP обрабатывает HTTP-запрос health-check.
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
	json.NewEncoder(w).Encode(Response{Status: status})
}
