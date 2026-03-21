package models

import "time"

type TextData struct {
	ID        int64
	UserID    int64
	Content   string
	Name      string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
