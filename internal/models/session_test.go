package models

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	s := &Session{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !s.IsExpired() {
		t.Fatal("expected session to be expired")
	}

	s2 := &Session{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if s2.IsExpired() {
		t.Fatal("expected session to not be expired")
	}
}

func TestSessionIsRevoked(t *testing.T) {
	now := time.Now()
	s := &Session{RevokedAt: &now}
	if !s.IsRevoked() {
		t.Fatal("expected session to be revoked")
	}

	s2 := &Session{RevokedAt: nil}
	if s2.IsRevoked() {
		t.Fatal("expected session to not be revoked")
	}
}

func TestSessionIsActive(t *testing.T) {
	s := &Session{
		ExpiresAt: time.Now().Add(1 * time.Hour),
		RevokedAt: nil,
	}
	if !s.IsActive() {
		t.Fatal("expected session to be active")
	}

	// Истёкшая сессия
	s2 := &Session{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		RevokedAt: nil,
	}
	if s2.IsActive() {
		t.Fatal("expected expired session to be inactive")
	}

	// Отозванная сессия
	now := time.Now()
	s3 := &Session{
		ExpiresAt: time.Now().Add(1 * time.Hour),
		RevokedAt: &now,
	}
	if s3.IsActive() {
		t.Fatal("expected revoked session to be inactive")
	}
}
