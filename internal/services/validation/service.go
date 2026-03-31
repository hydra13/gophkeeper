package validation

// ValidationService проверяет корректность входных данных.
type ValidationService interface {
	Validate(data interface{}) error
}
