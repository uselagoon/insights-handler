package routes

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/uselagoon/lagoon/services/insights-handler/internal/api/database"
	models "github.com/uselagoon/lagoon/services/insights-handler/internal/api/models"
)

type FactsRequest struct {
	Project     string            `json:"project"`
	Environment string            `json:"environment"`
	Facts       []models.Fact     `json:"facts,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
}

type ProblemsRequest struct {
	Project     string           `json:"project"`
	Environment string           `json:"environment"`
	Problems    []models.Problem `json:"problems"`
}

type DeploymentMetricsRequest struct {
	Project string `json:"project"`
}

type CreateDeploymentMetricsRequest struct {
	Project string `json:"project"`
}

func RegisterRoutes(router *gin.Engine, db *db.DBConnection) {
	router.GET("/facts", func(c *gin.Context) {
		getFactsHandler(c, db)
	})
	router.POST("/facts", func(c *gin.Context) {
		createFactsHandler(c, db)
	})

	router.GET("/problems", func(c *gin.Context) {
		getProblemsHandler(c, db)
	})
	router.POST("/problems", func(c *gin.Context) {
		createProblemsHandler(c, db)
	})

	router.GET("/deployments", func(c *gin.Context) {
		getDeploymentMetricsHandler(c, db)
	})
	router.POST("/deployments", func(c *gin.Context) {
		createDeploymentMetricsHandler(c, db)
	})
}

func getFactsHandler(c *gin.Context, db *db.DBConnection) {
	facts, err := db.GetFacts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve facts from the database",
		})
		return
	}

	c.JSON(http.StatusOK, facts)
}

func createFactsHandler(c *gin.Context, db *db.DBConnection) {
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

	err = db.InsertFacts(request.Facts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert facts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook received and processed successfully"})
}

func getProblemsHandler(c *gin.Context, db *db.DBConnection) {
	problems, err := db.GetProblems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve problems from the database",
		})
		fmt.Println(err.Error())
		return
	}

	c.JSON(http.StatusOK, problems)
}

func createProblemsHandler(c *gin.Context, db *db.DBConnection) {
	var request ProblemsRequest

	err := c.BindJSON(&request)
	if err != nil {
		fmt.Println("Error binding JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	log.Println("Received the following problems payload:")
	fmt.Println("Project:", request.Project)
	fmt.Println("Environment:", request.Environment)
	fmt.Println("Problems:")
	for _, problem := range request.Problems {
		fmt.Println("  Identifier:", problem.Identifier)
		fmt.Println("  Value:", problem.Value)
		fmt.Println("  Environment:", problem.Environment)
		fmt.Println("  Source:", problem.Source)
		fmt.Println("  Description:", problem.Description)
		fmt.Println("  Category:", problem.Category)
		fmt.Println("  KeyFact:", problem.KeyFact)
		fmt.Println()
	}

	err = db.InsertProblems(request.Problems)
	if err != nil {
		fmt.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert problems"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook received and processed successfully"})
}

func getDeploymentMetricsHandler(c *gin.Context, db *db.DBConnection) {
	var request DeploymentMetricsRequest

	err := c.BindJSON(&request)

	rows, err := db.GetDeploymentMetrics(request.Project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deployment metrics"})
		return
	}
	defer rows.Close()

	deploymentMetrics := models.DeploymentMetrics{
		Project:          request.Project,
		NumberOfBuilds:   100,
		FailedBuilds:     20,
		SuccessfulBuilds: 80,
		BuildsPerMonth:   25,
	}

	fmt.Println(deploymentMetrics)
}
