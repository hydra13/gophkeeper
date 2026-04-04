package validation

type ValidationService interface {
	Validate(data interface{}) error
}
