package models

// StoredPayload описывает сохранённую версию бинарного payload.
type StoredPayload struct {
	RecordID    int64
	Version     int64
	StoragePath string
	Size        int64
}
