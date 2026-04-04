package passwords

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hash string, password string) error {
	if hash == "" || password == "" {
		return errors.New("hash and password are required")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
