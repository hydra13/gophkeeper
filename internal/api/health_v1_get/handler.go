//go:generate minimock -i .HealthChecker -o mocks -s _mock.go -g
package health_v1_get

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	Status string `json:"status"`
}

type HealthChecker interface {
	Health() error
}

type Handler struct {
	checker HealthChecker
}

func NewHandler(checker HealthChecker) *Handler {
	return &Handler{checker: checker}
}

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
