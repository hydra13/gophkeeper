// Package uploads_by_id_v1_get реализует HTTP-ручку получения статуса upload-сессии.
//
// GET /api/v1/uploads/{id}
//
//go:generate minimock -i .UploadStatusGetter -o mocks -s _mock.go -g
package uploads_by_id_v1_get

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/hydra13/gophkeeper/internal/models"
)

// UploadStatusGetter — интерфейс сервиса получения статуса upload-сессии.
type UploadStatusGetter interface {
	// GetUploadStatus возвращает статус upload-сессии по ID.
	GetUploadStatus(uploadID int64) (*models.UploadStatusResponse, error)
}

// Handler обрабатывает GET /api/v1/uploads/{id}.
type Handler struct {
	service UploadStatusGetter
}

// NewHandler создаёт новый обработчик получения статуса загрузки.
func NewHandler(service UploadStatusGetter) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает статус upload-сессии и недостающие чанки.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uploadID, err := extractUploadID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid upload_id", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetUploadStatus(uploadID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// extractUploadID извлекает upload_id из URL path /api/v1/uploads/{id}.
func extractUploadID(path string) (int64, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "uploads" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrRange
}
