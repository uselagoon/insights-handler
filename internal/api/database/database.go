package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
	models "github.com/uselagoon/lagoon/services/insights-handler/internal/api/models"
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

// GetFacts retrieves all facts from the database
func (dbc *DBConnection) GetFacts() ([]models.Fact, error) {
	// Assuming you have a SQL database connection stored in db.SQLDB
	rows, err := dbc.db.Query("SELECT id, name FROM facts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	facts := []models.Fact{}
	for rows.Next() {
		var fact models.Fact
		err := rows.Scan(&fact.ID, &fact.Name)
		if err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return facts, nil
}

func (dbc *DBConnection) InsertFacts(facts []models.Fact) error {
	log.Println("Inserting facts into the database:", facts)

	stmt, err := dbc.db.Prepare("INSERT INTO facts (name, value, environment, source, description, category, key_fact, type) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")
	if err != nil {
		return fmt.Errorf("failed to prepare INSERT statement: %w", err)
	}

	for _, fact := range facts {
		_, err := stmt.Exec(fact.Name, fact.Value, fact.Environment, fact.Source, fact.Description, fact.Category, fact.KeyFact, fact.Type)
		if err != nil {
			return fmt.Errorf("failed to insert fact: %w", err)
		}
	}

	return nil
}

// GetProblems retrieves all problems from the database
func (dbc *DBConnection) GetProblems() ([]models.Problem, error) {
	rows, err := dbc.db.Query("SELECT id, identifier, description FROM problems")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	problems := []models.Problem{}
	for rows.Next() {
		var problem models.Problem
		err := rows.Scan(&problem.ID, &problem.Identifier, &problem.Description)
		if err != nil {
			return nil, err
		}
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return problems, nil
}

// InsertProblems inserts a slice of problems into the database
func (dbc *DBConnection) InsertProblems(problems []models.Problem) error {

	stmt, err := dbc.db.Prepare("INSERT INTO problems (identifier, value, environment, source, description, category, key_fact) VALUES ($1, $2, $3, $4, $5, $6, $7)")
	if err != nil {
		return fmt.Errorf("failed to prepare INSERT statement: %w", err)
	}

	for _, problem := range problems {
		_, err := stmt.Exec(problem.Identifier, problem.Value, problem.Environment, problem.Source, problem.Description, problem.Category, problem.KeyFact)
		if err != nil {
			return fmt.Errorf("failed to insert problem: %w", err)
		}
	}

	return nil
}
