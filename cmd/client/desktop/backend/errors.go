package backend

import (
	"errors"
	"strings"

	"github.com/hydra13/gophkeeper/internal/models"
)

func normalizeError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, models.ErrInvalidCredentials):
		return errors.New("invalid email or password")
	case errors.Is(err, models.ErrEmailAlreadyExists):
		return errors.New("email is already registered")
	case errors.Is(err, models.ErrUnauthorized):
		return errors.New("session is no longer valid, please login again")
	case errors.Is(err, models.ErrRecordNotFound):
		return errors.New("record not found")
	case errors.Is(err, models.ErrEmptyRecordName):
		return errors.New("record name is required")
	case errors.Is(err, models.ErrNilPayload):
		return errors.New("record payload is required")
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid credentials"):
		return errors.New("invalid email or password")
	case strings.Contains(msg, "email already exists"):
		return errors.New("email is already registered")
	case strings.Contains(msg, "offline: cannot sync"):
		return errors.New("sync is unavailable while offline")
	case strings.Contains(msg, "offline: cannot download"):
		return errors.New("binary download is unavailable while offline")
	case strings.Contains(msg, "TLS certificate is required"):
		return errors.New("TLS certificate is required to connect to the server")
	}

	return err
}
