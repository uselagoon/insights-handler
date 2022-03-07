package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"log"
	"strings"
)

func processImageInspectInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
	decoded, err := decodeGzipString(v)
	if err != nil {
		fmt.Errorf(err.Error())
	}

	_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		return nil, "", apiErr
	}

	source := fmt.Sprintf("image-inspect:%s", resource.Service)

	marshallDecoded, err := json.Marshal(decoded)
	var imageInspect ImageData

	err = json.Unmarshal(marshallDecoded, &imageInspect)
	if err != nil {
		return nil, "", err
	}

	facts, err := processFactsFromImageInspect(imageInspect, environment.Id, source)
	if err != nil {
		return nil, "", err
	}
	log.Printf("Successfully decoded image-inspect")

	facts, err = KeyFactsFilter(facts)
	if err != nil {
		return nil, "", err
	}

	return facts, source, nil
}

func processFactsFromImageInspect(imageInspectData ImageData, id int, source string) ([]LagoonFact, error) {
	var factsInput []LagoonFact

	var filteredFacts []EnvironmentVariable
	keyFactsExistMap := make(map[string]bool)

	// Check if image inspect contains useful environment variables
	if imageInspectData.Env != nil {
		for _, v := range imageInspectData.Env {
			var envSplitStr = strings.Split(v, "=")
			env := EnvironmentVariable{
				Key:   envSplitStr[0],
				Value: envSplitStr[1],
			}

			// Remove duplicate key facts
			if _, ok := keyFactsExistMap[env.Key]; !ok {
				keyFactsExistMap[env.Key] = true
				filteredFacts = append(filteredFacts, env)
			}
		}
	}

	for _, f := range filteredFacts {
		factsInput = append(factsInput, LagoonFact{
			Environment: id,
			Name:        f.Key,
			Value:       f.Value,
			Source:      source,
			Category:    "Environment Variable",
			KeyFact:     true,
			Type:        FactTypeText,
		})
	}
	return factsInput, nil
}
