package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDeleteFactsByEnvironmentEndpoint(t *testing.T) {
	type args struct {
		c             *gin.Context
		Facts         []lagoonclient.Fact
		EnvironmentID int
		Service       string
		Source        string
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Basic test",
			args: args{
				c:             nil,
				EnvironmentID: 1,
				Source:        "exampleSource",
				Facts: []lagoonclient.Fact{
					{
						Environment: 1,
						Name:        "TestFact1",
						Value:       "TestValue1",
						Source:      "exampleSource",
					},
					{
						Environment: 1,
						Name:        "TestFact2",
						Value:       "TestValue2",
						Source:      "exampleSource",
					},
					{
						Environment: 1,
						Name:        "TestFact3",
						Value:       "TestValue3",
						Source:      "differentSource",
					},
				},
			},
		},
	}

	db, err := SetUpDatabase(Dboptions{Filename: ":memory:"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	r, err := SetupRouter(db)

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Insert facts into the database
			for _, e := range tt.args.Facts {
				db.Create(&e)
			}

			// Create a mock HTTP request to the endpoint
			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/environment/%v/facts?source=%v", tt.args.EnvironmentID, tt.args.Source), nil)
			assert.NoError(t, err)

			// Create a mock HTTP response recorder
			w := httptest.NewRecorder()

			// Call the handler function
			r.ServeHTTP(w, req)

			// Check the HTTP status code
			assert.Equal(t, http.StatusOK, w.Code)

			// Query the database to check if the facts were deleted
			var remainingFacts []lagoonclient.Fact
			err = db.Where("environment = ?", tt.args.EnvironmentID).Find(&remainingFacts).Error
			assert.NoError(t, err)

			// Check if the expected facts were deleted
			for _, e := range tt.args.Facts {
				deleted := true
				// Check if the fact is still present in the database
				for _, remaining := range remainingFacts {
					if e.Name == remaining.Name && e.Source == tt.args.Source { // the second prop of the || ensures that only some of the items remain based on service
						deleted = false
					}
				}
				if !deleted {
					t.Errorf("Fact '%v' should have been deleted, but it still remains in the database", e.Name)
				}
			}
		})
	}
}

func TestPostFactsByEnvironmentEndpoint(t *testing.T) {
	type args struct {
		c             *gin.Context
		Facts         []lagoonclient.Fact
		EnvironmentID int
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Basic test",
			args: args{
				c:             nil,
				EnvironmentID: 1,
				Facts: []lagoonclient.Fact{
					{
						Environment: 1,
						Name:        "TestFact1",
						Value:       "TestValue1",
					},
					{
						Environment: 1,
						Name:        "TestFact2",
						Value:       "TestValue2",
					},
				},
			},
		},
	}

	db, err := SetUpDatabase(Dboptions{Filename: ":memory:"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	r, err := SetupRouter(db)

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP request to the endpoint
			reqBody, err := json.Marshal(tt.args.Facts)
			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/environment/%v/facts", tt.args.EnvironmentID), bytes.NewBuffer(reqBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create a mock HTTP response recorder
			w := httptest.NewRecorder()

			// Call the handler function
			r.ServeHTTP(w, req)

			// Check the HTTP status code
			assert.Equal(t, http.StatusCreated, w.Code)

			// Decode the response body
			var createdFacts []lagoonclient.Fact
			err = json.Unmarshal(w.Body.Bytes(), &createdFacts)
			assert.NoError(t, err)

			// Check if the created facts match the expected facts
			for _, e := range tt.args.Facts {
				appears := false
				//we test whether each of them appear in the result
				for _, testE := range createdFacts {
					if e.Name == testE.Name {
						appears = true
					}
				}
				if appears == false {
					t.Errorf("Fact '%v' does not appear in results", e.Name)
				}
			}
		})
	}
}

func TestGetFactsByEnvironmentEndpoint(t *testing.T) {
	type args struct {
		c             *gin.Context
		Facts         []lagoonclient.Fact
		EnvironmentId int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Basic test",
			args: args{
				c:             nil,
				EnvironmentId: 1,
				Facts: []lagoonclient.Fact{
					{
						Environment: 1,
						Name:        "TestFact1",
						Value:       "TestValue1",
					},
					{
						Environment: 1,
						Name:        "TestFact2",
						Value:       "TestValue2",
					},
				},
			},
		},
	}

	db, err := SetUpDatabase(Dboptions{Filename: ":memory:"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	r, err := SetupRouter(db)

	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	for _, tt := range tests {

		for _, e := range tt.args.Facts {
			db.Create(&e)
		}

		t.Run(tt.name, func(t *testing.T) {

			// Create a mock HTTP request to the endpoint
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/environment/%v/facts", tt.args.EnvironmentId), nil)
			assert.NoError(t, err)

			// Create a mock HTTP response recorder
			w := httptest.NewRecorder()

			// Call the handler function
			r.ServeHTTP(w, req)

			// Check the HTTP status code
			assert.Equal(t, http.StatusOK, w.Code)

			// Decode the response body
			var facts []lagoonclient.Fact
			err = json.Unmarshal(w.Body.Bytes(), &facts)
			assert.NoError(t, err)

			for _, e := range tt.args.Facts {
				appears := false
				//we test whether each of them appear in the result
				for _, testE := range facts {
					if e.Name == testE.Name {
						appears = true
					}
				}
				if appears == false {
					t.Errorf("Fact '%v' does not appear in results", e.Name)
				}
			}

		})
	}
}
