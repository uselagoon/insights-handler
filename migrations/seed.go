package migrations

import (
	"database/sql"
	"fmt"

	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
)

func RunSeed() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fmt.Errorf("failed to open database connection: %w", err)
	}

	err = insertSeedFacts(db)
	if err != nil {
		return err
	}

	return nil
}

func insertSeedFacts(db *sql.DB) error {
	_, err := db.Exec("INSERT INTO facts (name, value, environment, source) VALUES ('Fact 1', 'Value 1', '3', 'source') RETURNING *")
	if err != nil {
		return fmt.Errorf("failed to insert seed fact: %w", err)
	}

	_, err = db.Exec("INSERT INTO facts (name, value, environment, source) VALUES ('Fact 2', 'Value 2', '3', 'source') RETURNING *")
	if err != nil {
		return fmt.Errorf("failed to insert seed fact: %w", err)
	}

	return nil
}
