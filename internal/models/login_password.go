package models

// LoginPayload хранит логин и пароль записи.
type LoginPayload struct {
	Login    string
	Password string
}

// RecordType возвращает тип payload.
func (p LoginPayload) RecordType() RecordType {
	return RecordTypeLogin
}
