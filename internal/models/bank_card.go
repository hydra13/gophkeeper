package models

import "time"

type BankCard struct {
	ID           int64
	UserID       int64
	Number       string
	HolderName   string
	ExpiryDate   string
	CVV          string
	Name         string
	Metadata     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
