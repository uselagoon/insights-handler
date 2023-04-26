package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Khan/genqlient/graphql"
)

type FactsPayload struct {
	Facts []LagoonFact `json:"facts,omitempty"`
}

func processFactsInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
	if insights.LagoonType == Facts && insights.InsightsType == Raw {
		r := strings.NewReader(v)

		// Decode base64
		//dec := base64.NewDecoder(base64.StdEncoding, r)
		res, err := ioutil.ReadAll(r)
		if err != nil {
			fmt.Println("err: ", err)
		}

		source := fmt.Sprintf("insights:facts:%s", resource.Service)

		facts := processFactsFromJSON(res, source)
		facts, err = KeyFactsFilter(facts)
		if err != nil {
			return nil, "", err
		}

		if len(facts) == 0 {
			return nil, "", fmt.Errorf("no facts to process")
		}

		log.Printf("Successfully processed facts")
		log.Printf("- Facts found: %d\n", len(facts))

		return facts, source, nil
	}
	return nil, "", nil
}

func processFactsFromJSON(facts []byte, source string) []LagoonFact {
	var factsInput []LagoonFact

	var factsPayload FactsPayload
	err := json.Unmarshal(facts, &factsPayload)
	if err != nil {
		fmt.Println(err.Error())
		panic("Can't unmarshal facts")
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
		fmt.Println("Processing fact name " + f.Name)
		fact, _ = ProcessLagoonFactAgainstRegisteredFilters(fact, f)
		factsInput = append(factsInput, fact)
	}
	return factsInput
}

func init() {
	RegisterParserFilter(processFactsInsightsData)
}
