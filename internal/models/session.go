package models

import "time"

// Session описывает пользовательскую сессию на устройстве.
type Session struct {
	ID           int64
	UserID       int64
	DeviceID     string
	DeviceName   string
	ClientType   string
	RefreshToken string
	LastSeenAt   time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	CreatedAt    time.Time
}

// IsExpired сообщает, что срок действия сессии истёк.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRevoked сообщает, что сессия отозвана.
func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

// IsActive сообщает, что сессия ещё действительна.
func (s *Session) IsActive() bool {
	return !s.IsExpired() && !s.IsRevoked()
}
