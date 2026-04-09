package records_common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hydra13/gophkeeper/internal/api/responses"
	"github.com/hydra13/gophkeeper/internal/models"
)

func TestWriteConflict(t *testing.T) {
	t.Run("writes HTTP 409 with nil revisions", func(t *testing.T) {
		rec := httptest.NewRecorder()

		WriteConflict(rec, "revision conflict", nil, nil)

		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var resp ConflictResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, "revision conflict", resp.Error)
		require.Nil(t, resp.Conflict)
	})

	t.Run("writes HTTP 409 with non-nil revisions", func(t *testing.T) {
		rec := httptest.NewRecorder()

		local := int64(5)
		server := int64(10)

		WriteConflict(rec, "revision conflict", &local, &server)

		require.Equal(t, http.StatusConflict, rec.Code)

		var resp ConflictResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, "revision conflict", resp.Error)
		require.NotNil(t, resp.Conflict)
		require.Equal(t, int64(5), *resp.Conflict.LocalRevision)
		require.Equal(t, int64(10), *resp.Conflict.ServerRevision)
	})

	t.Run("writes HTTP 409 with only local revision", func(t *testing.T) {
		rec := httptest.NewRecorder()

		local := int64(3)

		WriteConflict(rec, "conflict", &local, nil)

		require.Equal(t, http.StatusConflict, rec.Code)

		var resp ConflictResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Conflict)
		require.Equal(t, int64(3), *resp.Conflict.LocalRevision)
		require.Nil(t, resp.Conflict.ServerRevision)
	})

	t.Run("writes HTTP 409 with only server revision", func(t *testing.T) {
		rec := httptest.NewRecorder()

		server := int64(7)

		WriteConflict(rec, "conflict", nil, &server)

		require.Equal(t, http.StatusConflict, rec.Code)

		var resp ConflictResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Conflict)
		require.Nil(t, resp.Conflict.LocalRevision)
		require.Equal(t, int64(7), *resp.Conflict.ServerRevision)
	})
}

func TestMapRecordError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedBody   string
		expectHandled  bool
		isConflict     bool
	}{
		{
			name:           "ErrRecordNotFound returns 404",
			err:            models.ErrRecordNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "record not found",
			expectHandled:  true,
		},
		{
			name:           "ErrAlreadyDeleted returns 400",
			err:            models.ErrAlreadyDeleted,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrAlreadyDeleted.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrRevisionConflict returns 409",
			err:            models.ErrRevisionConflict,
			expectedStatus: http.StatusConflict,
			expectedBody:   "revision conflict",
			expectHandled:  true,
			isConflict:     true,
		},
		{
			name:           "ErrRevisionNotMonotonic returns 400",
			err:            models.ErrRevisionNotMonotonic,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrRevisionNotMonotonic.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrEmptyRecordName returns 400",
			err:            models.ErrEmptyRecordName,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrEmptyRecordName.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrEmptyDeviceID returns 400",
			err:            models.ErrEmptyDeviceID,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrEmptyDeviceID.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrInvalidRecordType returns 400",
			err:            models.ErrInvalidRecordType,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrInvalidRecordType.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrNilPayload returns 400",
			err:            models.ErrNilPayload,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrNilPayload.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrInvalidKeyVersion returns 400",
			err:            models.ErrInvalidKeyVersion,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrInvalidKeyVersion.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrInvalidPayloadVersion returns 400",
			err:            models.ErrInvalidPayloadVersion,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrInvalidPayloadVersion.Error(),
			expectHandled:  true,
		},
		{
			name:           "ErrInvalidUserID returns 400",
			err:            models.ErrInvalidUserID,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   models.ErrInvalidUserID.Error(),
			expectHandled:  true,
		},
		{
			name:          "unknown error returns false",
			err:           errors.New("some random error"),
			expectHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			handled := MapRecordError(rec, tt.err)

			require.Equal(t, tt.expectHandled, handled)

			if tt.expectHandled {
				require.Equal(t, tt.expectedStatus, rec.Code)
				require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

				if tt.isConflict {
					var resp ConflictResponse
					err := json.NewDecoder(rec.Body).Decode(&resp)
					require.NoError(t, err)
					require.Equal(t, tt.expectedBody, resp.Error)
				} else {
					var resp responses.ErrorResponse
					err := json.NewDecoder(rec.Body).Decode(&resp)
					require.NoError(t, err)
					require.Equal(t, tt.expectedBody, resp.Error)
				}
			}
		})
	}
}
