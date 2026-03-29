package models

// LoginPayload — данные типа логин/пароль.
type LoginPayload struct {
	Login    string
	Password string
}

// RecordType возвращает тип записи login.
func (p LoginPayload) RecordType() RecordType {
	return RecordTypeLogin
}
