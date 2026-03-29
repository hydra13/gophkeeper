package validation

// ValidationService интерфейс валидации
type ValidationService interface {
	Validate(data interface{}) error
}
