package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultAccessTTL  = 15 * time.Minute
	refreshTokenBytes = 32
)

// Claims — JWT claims для access-токена.
type Claims struct {
	jwt.RegisteredClaims
	UserID    int64 `json:"uid"`
	SessionID int64 `json:"sid"`
}

// JWTManager генерирует и валидирует JWT access-токены и refresh-токены.
type JWTManager struct {
	secret    []byte
	accessTTL time.Duration
}

// NewJWTManager создаёт JWTManager с секретом и временем жизни access-токена.
func NewJWTManager(secret string, accessTTL time.Duration) (*JWTManager, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is required")
	}
	if accessTTL <= 0 {
		accessTTL = defaultAccessTTL
	}
	return &JWTManager{
		secret:    []byte(secret),
		accessTTL: accessTTL,
	}, nil
}

// NewAccessToken создаёт подписанный JWT access-токен для userID с привязкой к sessionID.
func (m *JWTManager) NewAccessToken(userID, sessionID int64) (string, error) {
	now := time.Now()
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

// NewRefreshToken генерирует криптографически случайный refresh-токен.
func (m *JWTManager) NewRefreshToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateToken парсит и валидирует access-токен, возвращает userID и sessionID.
func (m *JWTManager) ValidateToken(tokenString string) (int64, int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
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
