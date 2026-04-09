package models

// TextPayload хранит произвольный текст записи.
type TextPayload struct {
	Content string
}

// RecordType возвращает тип payload.
func (p TextPayload) RecordType() RecordType {
	return RecordTypeText
}
