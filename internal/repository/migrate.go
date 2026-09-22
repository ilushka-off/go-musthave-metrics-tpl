package repository

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	models "github.com/ilushka-off/go-musthave-metrics-tpl/internal/model"
)

// RunMigrations applies every pending schema migration embedded in
// models.FS to db. It is idempotent: running it again with no pending
// migrations is not an error.
func RunMigrations(db *sql.DB) error {
	sourceDriver, err := iofs.New(models.FS, "migrations")
	if err != nil {
		return err
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
