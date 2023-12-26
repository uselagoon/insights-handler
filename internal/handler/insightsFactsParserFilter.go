package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"strings"

	"github.com/Khan/genqlient/graphql"
)

type FactsPayload struct {
	Facts []LagoonFact `json:"facts,omitempty"`
}

// Processes facts from insights payloads that come from reconcilled kubernetes payloads (e.g. include labels/annotations and compressed/encoded data)
func processFactsInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
	source := fmt.Sprintf("insights:facts:%s", resource.Service)
	logger := slog.With("ProjectName", resource.Project, "EnvironmentName", resource.Environment, "Source", source)
	if insights.LagoonType == Facts && insights.InsightsType == Raw {
		r := strings.NewReader(v)

		// Decode base64
		//dec := base64.NewDecoder(base64.StdEncoding, r)
		res, err := ioutil.ReadAll(r)
		if err != nil {
			slog.Error("Error reading insights data", "Error", err)
		}

		facts := processFactsFromJSON(logger, res, source)
		facts, err = KeyFactsFilter(facts)
		if err != nil {
			return nil, "", err
		}

		if len(facts) == 0 {
			return nil, "", fmt.Errorf("no facts to process")
		}

		//log.Printf("Successfully processed %d fact(s), for '%s:%s', from source '%s'", len(facts), resource.Project, resource.Environment, source)
		logger.Info("Successfully processed facts", "number", len(facts))

		return facts, source, nil
	}
	return nil, "", nil
}

func processFactsFromJSON(logger *slog.Logger, facts []byte, source string) []LagoonFact {
	var factsInput []LagoonFact

	var factsPayload FactsPayload
	err := json.Unmarshal(facts, &factsPayload)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	if len(factsPayload.Facts) == 0 {
		return factsInput
	}

	var filteredFacts []LagoonFact
	keyFactsExistMap := make(map[string]bool)

	// Filter key facts
	for _, v := range factsPayload.Facts {
		if _, ok := keyFactsExistMap[v.Name]; !ok {
			keyFactsExistMap[v.Name] = true
			filteredFacts = append(filteredFacts, v)
		}
	}

	for _, f := range filteredFacts {
		fact := LagoonFact{
			Environment: f.Environment,
			Name:        f.Name,
			Value:       f.Value,
			Source:      source,
			Description: f.Description,
			KeyFact:     f.KeyFact,
			Type:        FactTypeText,
		}
		logger.Debug("Processing fact", "name", f.Name, "value", f.Value)
		fact, _ = ProcessLagoonFactAgainstRegisteredFilters(fact, f)
		factsInput = append(factsInput, fact)
	}
	return factsInput
}

func init() {
	RegisterParserFilter(processFactsInsightsData)
}
