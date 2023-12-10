package service

import (
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"gorm.io/gorm"
)

func CreateFacts(db *gorm.DB, facts *[]lagoonclient.Fact) error {
	return db.Create(facts).Error
}
