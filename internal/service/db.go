package service

import (
	"errors"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"gorm.io/gorm"
)

func CreateFacts(db *gorm.DB, facts *[]lagoonclient.Fact) error {
	return db.Create(facts).Error
}

type DeleteFactsOptions struct {
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
