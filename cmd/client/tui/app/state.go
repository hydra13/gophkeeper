package app

import "github.com/hydra13/gophkeeper/internal/models"

type state struct {
	filter     models.RecordType
	records    []models.Record
	selectedID int64
}
