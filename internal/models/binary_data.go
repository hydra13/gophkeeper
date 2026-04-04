package models

// BinaryPayload представляет бинарную запись.
type BinaryPayload struct {
	Data []byte
}

// RecordType возвращает тип payload.
func (p BinaryPayload) RecordType() RecordType {
	return RecordTypeBinary
}
