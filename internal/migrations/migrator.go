// Package migrations применяет встроенные SQL-миграции сервиса.
package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"

	appmigrations "github.com/hydra13/gophkeeper/migrations"
)

// Apply применяет все встроенные SQL-миграции к указанной базе данных.
func Apply(db *sql.DB) error {
	if err := goose.SetDialect("pgx"); err != nil {
		return err
	}
	goose.SetBaseFS(appmigrations.EmbedMigrations)
	return goose.Up(db, ".")
}
