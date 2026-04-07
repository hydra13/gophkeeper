package records_common

import (
	"errors"
	"net/http"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

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

// WriteConflict пишет ответ о конфликте ревизий.
func WriteConflict(w http.ResponseWriter, message string, localRevision, serverRevision *int64) {
	var conflict *ConflictInfo
	if localRevision != nil || serverRevision != nil {
		conflict = &ConflictInfo{
			LocalRevision:  localRevision,
			ServerRevision: serverRevision,
		}
	}

	responses.JSON(w, http.StatusConflict, ConflictResponse{Error: message, Conflict: conflict})
}

// MapRecordError преобразует доменную ошибку записи в HTTP-ответ.
func MapRecordError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, models.ErrRecordNotFound):
		responses.Error(w, http.StatusNotFound, "record not found")
	case errors.Is(err, models.ErrAlreadyDeleted):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrRevisionConflict):
		WriteConflict(w, "revision conflict", nil, nil)
	case errors.Is(err, models.ErrRevisionNotMonotonic):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrEmptyRecordName):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrEmptyDeviceID):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidRecordType):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrNilPayload):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidKeyVersion):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidPayloadVersion):
		responses.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, models.ErrInvalidUserID):
		responses.Error(w, http.StatusBadRequest, err.Error())
	default:
		return false
	}
	return true
}
