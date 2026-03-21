package crypto

// CryptoService интерфейс шифрования
type CryptoService interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}
