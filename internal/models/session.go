package models

import "time"

// Session — device-aware пользовательская сессия.
// Каждое устройство получает отдельную сессию с собственным refresh-токеном.
type Session struct {
	// ID — уникальный идентификатор сессии.
	ID int64
	// UserID — владелец сессии.
	UserID int64
	// DeviceID — идентификатор устройства клиента.
	DeviceID string
	// DeviceName — понятное имя устройства (например, "MacBook Pro").
	DeviceName string
	// ClientType — тип клиента (cli, desktop, web).
	ClientType string
	// RefreshToken — токен обновления access-токена.
	RefreshToken string
	// LastSeenAt — время последней активности сессии.
	LastSeenAt time.Time
	// ExpiresAt — время истечения сессии.
	ExpiresAt time.Time
	// RevokedAt — время отзыва сессии; nil означает активную сессию.
	RevokedAt *time.Time
	// CreatedAt — время создания сессии.
	CreatedAt time.Time
}

// IsExpired проверяет, истекла ли сессия.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked проверяет, отозвана ли сессия.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsActive проверяет, активна ли сессия (не истекла и не отозвана).
func (s *Session) IsActive() bool {
	return !s.IsExpired() && !s.IsRevoked()
}
