package records_common

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/models"
)

// ErrorResponse описывает ответ с ошибкой.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ConflictInfo описывает детали конфликта ревизий.
type ConflictInfo struct {
	LocalRevision  *int64 `json:"local_revision,omitempty"`
	ServerRevision *int64 `json:"server_revision,omitempty"`
}

// ConflictResponse описывает ответ с конфликтом ревизий.
type ConflictResponse struct {
	Error    string        `json:"error"`
	Conflict *ConflictInfo `json:"conflict,omitempty"`
}

// WriteError пишет ошибку API записи.
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		log.Printf("record error response encode failed: %v", err)
	}
}

// WriteConflict пишет ответ о конфликте ревизий.
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
	if err := json.NewEncoder(w).Encode(ConflictResponse{Error: message, Conflict: conflict}); err != nil {
		log.Printf("record conflict response encode failed: %v", err)
	}
}

// MapRecordError преобразует доменную ошибку записи в HTTP-ответ.
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
