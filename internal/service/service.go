package service

import (
	"github.com/gin-gonic/gin"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func SetupRouter(db *gorm.DB) (*gin.Engine, error) {
	router := gin.Default()

	// Set up DB as middleware for injection
	router.Use(func(context *gin.Context) {
		context.Set("db", db)
		context.Next()
	})

	router.GET("/environment/:id/facts", GetFactsByEnvironmentEndpoint)
	router.GET("/ping/{id}/facts", func(context *gin.Context) {
		context.String(200, "pong")
	})
	return router, nil
}

type dboptions struct {
	Filename string
}

// setUpDatabase will connect to the selected DB and run pending migrations
func setUpDatabase(opts dboptions) (*gorm.DB, error) {
	// TODO: currently we're only supporting sqlite for dev
	// going forward, this will run on mysql - but both should be selected

	db, err := gorm.Open(sqlite.Open(opts.Filename), &gorm.Config{})
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

//func CreateProblem(c *gin.Context) {
//
//}
//
//func GetProblem(c *gin.Context) {
//
//}
//
//func GetProblemsForEnvironment(c *gin.Context) {
//
//}

func GetFactsByEnvironmentEndpoint(c *gin.Context) {
	// Retrieve the environmentID parameter from the URL

	environmentIDStr := c.Param("id")
	environmentID, err := strconv.Atoi(environmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environmentID"})
		return
	}

	// Retrieve the GORM database instance from the Gin context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
		return
	}

	// Cast the db interface to *gorm.DB
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid database connection"})
		return
	}

	// Query the database for facts by environmentID
	var facts []lagoonclient.Fact
	if err := gormDB.Where("environment = ?", environmentID).Find(&facts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve facts from the database"})
		return
	}

	// Return the facts as JSON
	c.JSON(http.StatusOK, facts)
}
