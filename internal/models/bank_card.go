package models

import (
	"fmt"
	"unicode"
)

// CardPayload хранит данные банковской карты.
type CardPayload struct {
	Number     string
	HolderName string
	ExpiryDate string
	CVV        string
}

// RecordType возвращает тип payload.
func (p CardPayload) RecordType() RecordType {
	return RecordTypeCard
}

// Validate проверяет формат полей карты.
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
