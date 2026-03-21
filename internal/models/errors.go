package models

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDataNotFound       = errors.New("data not found")
	ErrConflict           = errors.New("data conflict")
	ErrUnauthorized       = errors.New("unauthorized")
)
