package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/api/models"
)

type DBConnection struct {
	db *sql.DB
}

func NewDBConnection(config *config.Config) (*DBConnection, error) {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &DBConnection{
		db: db,
	}, nil
}

func (dbc *DBConnection) InsertFacts(facts []models.Fact) error {
	// Implement the logic to insert facts into the database
	// Use dbc.db to execute SQL statements and interact with the database
	// ...
	log.Println("Inserting facts into the database:", facts)

	return nil
}
