package migrations

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
)

func RunMigrations() error {
	cfg, err := config.LoadConfig("", "localhost", "postgres", "example", "5432")
	if err != nil {
		fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	err = goose.Down(db, "./migrations")
	if err != nil {
		return fmt.Errorf("failed to drop all migrations: %w", err)
	}

	// run any pending migrations found in this directory
	err = goose.Up(db, "./migrations")
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
