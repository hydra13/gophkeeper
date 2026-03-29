package passwords

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword создаёт безопасный bcrypt-хеш пароля.
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

// ComparePassword проверяет пароль по bcrypt-хешу.
func ComparePassword(hash string, password string) error {
	if hash == "" || password == "" {
		return errors.New("hash and password are required")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
