package models

import "time"

type LoginPassword struct {
	ID        int64
	UserID    int64
	Login     string
	Password  string
	Name      string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
