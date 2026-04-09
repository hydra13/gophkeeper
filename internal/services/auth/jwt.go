package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hydra13/gophkeeper/internal/option"
)

const (
	defaultAccessTTL  = 15 * time.Minute
	refreshTokenBytes = 32
)

// Claims описывает пользовательские claims access-токена.
type Claims struct {
	jwt.RegisteredClaims
	UserID    int64 `json:"uid"`
	SessionID int64 `json:"sid"`
}

// JWTManager выпускает и проверяет JWT и refresh-токены.
type JWTManager struct {
	secret    []byte
	accessTTL time.Duration
	now       func() time.Time
}

// JWTOption настраивает optional/runtime/test-параметры JWT manager.
type JWTOption = option.Option[JWTManager]

// WithAccessTTL задаёт TTL access-токена.
func WithAccessTTL(ttl time.Duration) JWTOption {
	return func(m *JWTManager) {
		m.accessTTL = ttl
	}
}

// WithJWTClock подменяет источник времени для тестов и runtime-настроек.
func WithJWTClock(now func() time.Time) JWTOption {
	return func(m *JWTManager) {
		if now != nil {
			m.now = now
		}
	}
}

// NewJWTManager создаёт менеджер JWT с заданным TTL access-токена.
func NewJWTManager(secret string, opts ...JWTOption) (*JWTManager, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is required")
	}
	manager := &JWTManager{
		secret:    []byte(secret),
		accessTTL: defaultAccessTTL,
		now:       time.Now,
	}
	option.Apply(manager, opts...)
	if manager.accessTTL <= 0 {
		return nil, errors.New("access ttl must be positive")
	}
	return manager, nil
}

// NewAccessToken выпускает access-токен для пользователя и сессии.
func (m *JWTManager) NewAccessToken(userID, sessionID int64) (string, error) {
	now := m.now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
		UserID:    userID,
		SessionID: sessionID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// NewRefreshToken генерирует случайный refresh-токен.
func (m *JWTManager) NewRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateToken проверяет токен и возвращает userID и sessionID.
func (m *JWTManager) ValidateToken(tokenString string) (int64, int64, error) {
	parser := jwt.NewParser(jwt.WithTimeFunc(m.now))
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return 0, 0, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return 0, 0, errors.New("invalid token claims")
	}
	return claims.UserID, claims.SessionID, nil
}
