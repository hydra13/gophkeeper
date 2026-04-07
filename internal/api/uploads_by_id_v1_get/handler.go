//go:generate minimock -i .UploadStatusGetter -o mocks -s _mock.go -g
package uploads_by_id_v1_get

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

// UploadStatusGetter описывает получение статуса загрузки.
type UploadStatusGetter interface {
	GetUploadStatus(uploadID int64) (*models.UploadStatusResponse, error)
}

// Handler обрабатывает получение статуса загрузки.
type Handler struct {
	service UploadStatusGetter
}

// NewHandler создаёт обработчик статуса загрузки.
func NewHandler(service UploadStatusGetter) *Handler {
	return &Handler{service: service}
}

// ServeHTTP возвращает статус загрузки.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		responses.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	uploadID, err := extractUploadID(r.URL.Path)
	if err != nil {
		responses.Error(w, http.StatusBadRequest, "invalid upload_id")
		return
	}

	resp, err := h.service.GetUploadStatus(uploadID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			responses.Error(w, http.StatusNotFound, err.Error())
			return
		}
		responses.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.JSON(w, http.StatusOK, resp)
}

func extractUploadID(path string) (int64, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "uploads" && i+1 < len(parts) {
			return strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return 0, strconv.ErrRange
}
