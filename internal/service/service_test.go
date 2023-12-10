package service

import (
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

	db, err := setUpDatabase(dboptions{Filename: ":memory:"})
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
