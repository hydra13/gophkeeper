package models

import (
	"errors"
	"testing"
)

func TestCardPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		card    CardPayload
		wantErr error
	}{
		{
			name: "valid Visa 16 digits",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan Ivanov",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: nil,
		},
		{
			name: "valid Mastercard",
			card: CardPayload{
				Number:     "5500000000000004",
				HolderName: "John Doe",
				ExpiryDate: "01/26",
				CVV:        "000",
			},
			wantErr: nil,
		},
		{
			name: "valid Amex 15 digits with 4-digit CVV",
			card: CardPayload{
				Number:     "378282246310005",
				HolderName: "Test User",
				ExpiryDate: "06/25",
				CVV:        "1234",
			},
			wantErr: nil,
		},
		{
			name: "empty card number",
			card: CardPayload{
				Number:     "",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "card number too short",
			card: CardPayload{
				Number:     "4111111",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "card number too long",
			card: CardPayload{
				Number:     "41111111111111111111",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "card number with letters",
			card: CardPayload{
				Number:     "4111abcd11111111",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "card number fails Luhn",
			card: CardPayload{
				Number:     "4111111111111112",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidCardNumber,
		},
		{
			name: "empty holder name",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "",
				ExpiryDate: "12/28",
				CVV:        "123",
			},
			wantErr: ErrEmptyCardHolder,
		},
		{
			name: "expiry wrong format - no slash",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "1228",
				CVV:        "123",
			},
			wantErr: ErrInvalidExpiryDate,
		},
		{
			name: "expiry month 00",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "00/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidExpiryDate,
		},
		{
			name: "expiry month 13",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "13/28",
				CVV:        "123",
			},
			wantErr: ErrInvalidExpiryDate,
		},
		{
			name: "expiry with letters",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "ab/cd",
				CVV:        "123",
			},
			wantErr: ErrInvalidExpiryDate,
		},
		{
			name: "CVV too short",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "12",
			},
			wantErr: ErrInvalidCVV,
		},
		{
			name: "CVV too long",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "12345",
			},
			wantErr: ErrInvalidCVV,
		},
		{
			name: "CVV with letters",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "1a3",
			},
			wantErr: ErrInvalidCVV,
		},
		{
			name: "empty CVV",
			card: CardPayload{
				Number:     "4532015112830366",
				HolderName: "Ivan",
				ExpiryDate: "12/28",
				CVV:        "",
			},
			wantErr: ErrInvalidCVV,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.card.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestLuhnCheck(t *testing.T) {
	tests := []struct {
		number string
		valid  bool
	}{
		{"4532015112830366", true},  // Visa
		{"5500000000000004", true},  // Mastercard
		{"378282246310005", true},   // Amex
		{"4111111111111111", true},  // Visa test
		{"4111111111111112", false}, // Luhn fail
		{"0000000000000000", true},  // All zeros (technically valid Luhn)
		{"1234567890123456", false}, // Random invalid
	}

	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			got := luhnCheck(tt.number)
			if got != tt.valid {
				t.Errorf("luhnCheck(%q) = %v, want %v", tt.number, got, tt.valid)
			}
		})
	}
}
