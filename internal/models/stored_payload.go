package models

// StoredPayload описывает сохранённый бинарный payload в blob storage.
type StoredPayload struct {
	RecordID    int64
	Version     int64
	StoragePath string
	Size        int64
}
