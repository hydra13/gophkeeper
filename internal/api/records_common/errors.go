package recordscommon

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// Transport rules for records endpoints:
// 401 — unauthenticated, 403 — access denied, 404 — not found, 409 — revision conflict.

// ErrorResponse — единый DTO ошибок для records endpoint-ов.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ConflictInfo — DTO информации о конфликте ревизий.
type ConflictInfo struct {
	LocalRevision  *int64 `json:"local_revision,omitempty"`
	ServerRevision *int64 `json:"server_revision,omitempty"`
}

// ConflictResponse — DTO ответа о конфликте ревизий.
type ConflictResponse struct {
	Error    string        `json:"error"`
	Conflict *ConflictInfo `json:"conflict,omitempty"`
}

// WriteError записывает ErrorResponse с указанным статусом.
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// WriteConflict записывает ConflictResponse с HTTP 409.
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

// MapRecordError маппит доменные ошибки записей в HTTP-статусы и записывает ответ.
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
