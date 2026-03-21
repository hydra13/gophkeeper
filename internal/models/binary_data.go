package models

import "time"

type BinaryData struct {
	ID        int64
	UserID    int64
	Data      []byte
	Name      string
	Metadata  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
