package responses

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse описывает унифицированный JSON-ответ с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON пишет JSON-ответ с указанным статусом.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api response encode failed: %v", err)
	}
}

// Error пишет JSON-ответ с ошибкой.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{Error: message})
}

// NoContent пишет ответ без тела.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
