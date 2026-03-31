package recordscommon

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// Коды ответов для ручек записей:
// 401 — пользователь не аутентифицирован.
// 403 — доступ к записи запрещён.
// 404 — запись не найдена.
// 409 — конфликт ревизий.

// ErrorResponse — единый формат ошибки для ручек записей.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ConflictInfo содержит ревизии, участвующие в конфликте.
type ConflictInfo struct {
	LocalRevision  *int64 `json:"local_revision,omitempty"`
	ServerRevision *int64 `json:"server_revision,omitempty"`
}

// ConflictResponse описывает ошибку конфликта ревизий.
type ConflictResponse struct {
	Error    string        `json:"error"`
	Conflict *ConflictInfo `json:"conflict,omitempty"`
}

// WriteError записывает JSON-ответ с ошибкой и HTTP-статусом.
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// WriteConflict записывает JSON-ответ о конфликте ревизий с HTTP 409.
func WriteConflict(w http.ResponseWriter, message string, localRevision, serverRevision *int64) {
	var conflict *ConflictInfo
	if localRevision != nil || serverRevision != nil {
		conflict = &ConflictInfo{
			LocalRevision:  localRevision,
			ServerRevision: serverRevision,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(ConflictResponse{Error: message, Conflict: conflict})
}

// MapRecordError преобразует доменные ошибки записей в HTTP-ответы.
// Возвращает true если ошибка обработана.
func MapRecordError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, models.ErrRecordNotFound):
		WriteError(w, http.StatusNotFound, "record not found")
	case errors.Is(err, models.ErrAlreadyDeleted):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrRevisionConflict):
		WriteConflict(w, "revision conflict", nil, nil)
	case errors.Is(err, models.ErrRevisionNotMonotonic):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrEmptyRecordName):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrEmptyDeviceID):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidRecordType):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrNilPayload):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidKeyVersion):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidPayloadVersion):
		WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidUserID):
		WriteError(w, http.StatusBadRequest, err.Error())
	default:
		return false
	}
	return true
}
