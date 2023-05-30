package migrations

import (
	"database/sql"
	"fmt"

	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
)

func RunSeed() error {
	cfg, err := config.LoadConfig("", "localhost", "postgres", "example", "5432")
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
	_, err := db.Exec("INSERT INTO facts (name, value, environment, source, description, category, key_fact, type) VALUES ('Fact 1', 'Value 1', 3, 'source', 'Random description 1', 'Category 1', true, 'STRING') RETURNING *")
	if err != nil {
		return fmt.Errorf("failed to insert seed fact: %w", err)
	}

	_, err = db.Exec("INSERT INTO facts (name, value, environment, source, description, category, key_fact, type) VALUES ('Fact 2', 'Value 2', 3, 'source', 'Random description 2', 'Category 2', true, 'STRING') RETURNING *")

	if err != nil {
		return fmt.Errorf("failed to insert seed fact: %w", err)
	}

	return nil
}
