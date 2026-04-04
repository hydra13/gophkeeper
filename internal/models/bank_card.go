package models

import (
	"fmt"
	"unicode"
)

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

// Validate проверяет корректность данных банковской карты:
//   - Number: 13–19 цифр, проходит алгоритм Луна
//   - HolderName: непустое значение
//   - ExpiryDate: формат MM/YY, месяц 01–12
//   - CVV: 3 или 4 цифры
func (p CardPayload) Validate() error {
	if err := validateCardNumber(p.Number); err != nil {
		return err
	}
	if p.HolderName == "" {
		return ErrEmptyCardHolder
	}
	if err := validateExpiryDate(p.ExpiryDate); err != nil {
		return err
	}
	if err := validateCVV(p.CVV); err != nil {
		return err
	}
	return nil
}

// validateCardNumber проверяет что номер карты состоит из 13–19 цифр и проходит Luhn.
func validateCardNumber(number string) error {
	if len(number) < 13 || len(number) > 19 {
		return ErrInvalidCardNumber
	}
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return ErrInvalidCardNumber
		}
	}
	if !luhnCheck(number) {
		return ErrInvalidCardNumber
	}
	return nil
}

// luhnCheck проверяет номер карты по алгоритму Луна.
func luhnCheck(number string) bool {
	var sum int
	parity := len(number) % 2
	for i, r := range number {
		d := int(r - '0')
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

// validateExpiryDate проверяет формат MM/YY и что месяц в диапазоне 01–12.
func validateExpiryDate(expiry string) error {
	if len(expiry) != 5 {
		return ErrInvalidExpiryDate
	}
	if expiry[2] != '/' {
		return ErrInvalidExpiryDate
	}
	mm := expiry[0:2]
	yy := expiry[3:5]
	if !isDigits(mm) || !isDigits(yy) {
		return ErrInvalidExpiryDate
	}
	month := int(mm[0]-'0')*10 + int(mm[1]-'0')
	if month < 1 || month > 12 {
		return fmt.Errorf("%w: month must be 01-12", ErrInvalidExpiryDate)
	}
	return nil
}

// validateCVV проверяет что CVV состоит из 3 или 4 цифр.
func validateCVV(cvv string) error {
	if (len(cvv) != 3 && len(cvv) != 4) || !isDigits(cvv) {
		return ErrInvalidCVV
	}
	return nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
