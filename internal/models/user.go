package models

import "time"

// User — зарегистрированный пользователь системы.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID int64
	// Email — уникальный адрес электронной почты пользователя.
	Email string
	// PasswordHash — безопасный хеш пароля (bcrypt/argon2).
	PasswordHash string
	// CreatedAt — время регистрации.
	CreatedAt time.Time
	// UpdatedAt — время последнего обновления профиля.
	UpdatedAt time.Time
}
