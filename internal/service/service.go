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
	router.POST("/environment/:id/facts", PostFactsByEnvironmentEndpoint)
	router.DELETE("/environment/:id/facts", DeleteFactsByEnvironmentEndpoint)
	return router, nil
}

type Dboptions struct {
	Filename string
}

// SetUpDatabase will connect to the selected DB and run pending migrations
func SetUpDatabase(opts Dboptions) (*gorm.DB, error) {
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

// Fact represents the data model for a fact (assuming it's defined globally)

func DeleteFactsByEnvironmentEndpoint(c *gin.Context) {
	// Retrieve the environmentID parameter from the URL
	environmentIDStr := c.Param("id")
	environmentID, err := strconv.Atoi(environmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid environmentID"})
		return
	}

	// Retrieve optional parameters from query string
	source := c.Query("source")

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

	// Delete facts based on conditions
	rowsAffected, err := DeleteFacts(gormDB, environmentID, source) //gormDB.Where(conditions).Delete(&lagoonclient.Fact{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete facts from the database"})
		return
	}

	// Check if any records were deleted
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No matching facts found for deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Facts deleted successfully"})
}

func PostFactsByEnvironmentEndpoint(c *gin.Context) {
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

	// Bind the JSON request body to a slice of Fact structs
	var newFacts []lagoonclient.Fact
	if err := c.ShouldBindJSON(&newFacts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Set the environmentID for each new fact
	for i := range newFacts {
		newFacts[i].Environment = environmentID
	}

	// Create the new facts in the database
	if err := CreateFacts(gormDB, &newFacts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create facts in the database"})
		return
	}

	// Return the newly created facts as JSON
	c.JSON(http.StatusCreated, newFacts)
}

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
