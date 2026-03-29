package models

import "time"

// Metadata — произвольная текстовая метаинформация, привязанная к записи.
type Metadata struct {
	// ID — уникальный идентификатор метаданных.
	ID int64
	// RecordID — ссылка на запись, к которой привязана метаданные.
	RecordID int64
	// Key — ключ метаинформации.
	Key string
	// Value — значение метаинформации.
	Value string
	// CreatedAt — время создания.
	CreatedAt time.Time
}
