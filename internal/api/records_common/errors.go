package recordscommon

import (
	"encoding/json"
	"net/http"
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
