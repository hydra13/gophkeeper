package models

// CardPayload — данные банковской карты.
type CardPayload struct {
	Number     string
	HolderName string
	ExpiryDate string
	CVV        string
}

// RecordType возвращает тип записи card.
func (p CardPayload) RecordType() RecordType {
	return RecordTypeCard
}
