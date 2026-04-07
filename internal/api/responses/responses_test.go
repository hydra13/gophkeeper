package responses

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	JSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, map[string]string{"status": "ok"}, body)
}

func TestError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	Error(rec, http.StatusBadRequest, "invalid request body")

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, ErrorResponse{Error: "invalid request body"}, body)
}

func TestNoContent(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	NoContent(rec)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())
	require.Empty(t, rec.Header().Get("Content-Type"))
}
