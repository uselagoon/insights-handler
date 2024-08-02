package service

import (
	"database/sql"
	"errors"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Dboptions struct {
	Filename string
	DSN      string
}

// SetUpDatabase will connect to the selected DB and run pending migrations
func SetUpDatabase(opts Dboptions) (*gorm.DB, error) {
	// TODO: currently we're only supporting sqlite for dev
	// going forward, this will run on mysql - but both should be selected
	db := &gorm.DB{}
	var err error

	// set up the database connection based on the DbOptions | Filename - sqlite | DSN - postgres
	if opts.Filename != "" {
		db, err = gorm.Open(sqlite.Open(opts.Filename), &gorm.Config{})
	} else if opts.DSN != "" {
		pgDB, pgErr := sql.Open("pgx", opts.DSN)
		if pgErr != nil {
			return db, pgErr
		}
		db, err = gorm.Open(postgres.New(postgres.Config{
			Conn: pgDB,
		}), &gorm.Config{})
	}
	if err != nil {
		return db, err
	}

	if err = db.AutoMigrate(&lagoonclient.Fact{}); err != nil {
		return db, err
	}

	if err = db.AutoMigrate(&lagoonclient.LagoonProblem{}); err != nil {
		return db, err
	}

	return db, nil
}

func CreateFacts(db *gorm.DB, facts *[]lagoonclient.Fact) error {
	return db.Create(facts).Error
}

func DeleteFacts(db *gorm.DB, environmentId int, source string) (int64, error) {

	if environmentId == 0 {
		return 0, errors.New("EnvironmentId cannot be 0")
	}

	conditions := map[string]interface{}{
		"environment": environmentId,
	}

	if source != "" {
		conditions["source"] = source
	}

	result := db.Where(conditions).Delete(&lagoonclient.Fact{})

	return result.RowsAffected, result.Error
}

func GetFacts(db *gorm.DB, environmentId int) ([]lagoonclient.Fact, error) {
	var facts []lagoonclient.Fact
	res := db.Where("environment = ?", environmentId).Find(&facts)
	return facts, res.Error
}
