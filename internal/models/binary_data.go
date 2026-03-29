package models

// BinaryPayload — произвольные бинарные данные.
// Само содержимое передаётся через chunk upload/download с использованием UploadSession и DownloadSession.
type BinaryPayload struct {
	// Data — бинарное содержимое; может быть nil, если данные загружаются через upload session.
	Data []byte
}

// RecordType возвращает тип записи binary.
func (p BinaryPayload) RecordType() RecordType {
	return RecordTypeBinary
}
