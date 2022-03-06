package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"log"
	"strings"
)

func processImageInspectInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {

	if insights.OutputCompressed {
		decoded, err := decodeGzipString(v)
		if err != nil {
			fmt.Errorf(err.Error())
		}

		_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
		if apiErr != nil {
			return nil, "", apiErr
		}
		source := fmt.Sprintf("image-inspect:%s", resource.Service)
		log.Println(source)
		marshallDecoded, err := json.Marshal(decoded)
		var imageInspect ImageInspectData
		err = json.Unmarshal(marshallDecoded, &imageInspect)
		if err != nil {
			return nil, "", err
		}

		facts, err := processFactsFromImageInspect(imageInspect, environment.Id, source)
		if err != nil {
			return nil,"", err
		}
		log.Printf("Successfully decoded image-inspect")

		facts, err = keyFactsFilter(facts)
		if err != nil {
			return nil,"", err
		}

		return facts, source, nil
	}
	return []LagoonFact{}, "", errors.New("insights.OutputCompressed disabled - not processing incoming data")
}


func processFactsFromImageInspect(imageInspectData ImageInspectData, id int, source string) ([]LagoonFact, error) {
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
