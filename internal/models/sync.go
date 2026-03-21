package models

import "time"

type SyncRecord struct {
	ID        int64
	UserID    int64
	DataType  string
	DataID    int64
	Version   int64
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SyncRequest struct {
	UserID   int64
	Since    time.Time
	DataType string
}

type SyncResponse struct {
	Records []SyncRecord
}
