package routes

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/uselagoon/lagoon/services/insights-handler/internal/api/database"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/api/models"
)

type FactsRequest struct {
	Project     string            `json:"project,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Facts       []models.Fact     `json:"facts,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
}

func RegisterRoutes(router *gin.Engine, db *db.DBConnection) {
	router.GET("/facts", func(c *gin.Context) {
		// Handle GET request
	})

	router.POST("/facts", func(c *gin.Context) {
		var request FactsRequest
		err := c.BindJSON(&request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
			return
		}

		err = db.InsertFacts(request.Facts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert facts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Webhook received and processed successfully"})
	})
}

func getFactsHandler(c *gin.Context) {
}

func createFactsHandler(c *gin.Context) {
	var request FactsRequest
	err := c.BindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	log.Println("Recieved the following facts payload: ")
	fmt.Println("Project:", request.Project)
	fmt.Println("Environment:", request.Environment)
	fmt.Println("Facts:")
	for _, fact := range request.Facts {
		fmt.Println("  Name:", fact.Name)
		fmt.Println("  Value:", fact.Value)
		fmt.Println("  Environment:", fact.Environment)
		fmt.Println("  Source:", fact.Source)
		fmt.Println("  Description:", fact.Description)
		fmt.Println("  Category:", fact.Category)
		fmt.Println("  KeyFact:", fact.KeyFact)
		fmt.Println("  Type:", fact.Type)
		fmt.Println()
	}

	// access the extracted request data
	factsData := request.Facts
	fmt.Println(factsData)

	c.JSON(http.StatusOK, gin.H{"message": "Webhook received and processed successfully"})
}

// curl -X POST -H "Content-Type: application/json" -d '{"data": "123"}' http://localhost:8888/facts
