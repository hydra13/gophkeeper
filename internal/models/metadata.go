package models

import "time"

type Metadata struct {
	ID        int64
	DataType  string
	DataID    int64
	Key       string
	Value     string
	CreatedAt time.Time
}
