package models

// TextPayload — произвольные текстовые данные.
type TextPayload struct {
	Content string
}

// RecordType возвращает тип записи text.
func (p TextPayload) RecordType() RecordType {
	return RecordTypeText
}
