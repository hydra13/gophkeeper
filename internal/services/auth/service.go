package auth

// AuthService интерфейс аутентификации
type AuthService interface {
	Register(login, password string) error
	Login(login, password string) (string, error)
	ValidateToken(token string) (int64, error)
}
