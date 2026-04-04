package models

import "time"

// Metadata хранит дополнительные сведения о записи.
type Metadata struct {
	ID        int64
	RecordID  int64
	Key       string
	Value     string
	CreatedAt time.Time
}
