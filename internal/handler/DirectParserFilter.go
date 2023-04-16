package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Khan/genqlient/graphql"
)

//This becomes/implements the ParserFilter interface
type DirectInsightsData struct {
	Data map[string]struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	} `json:"data"`
}

func processDirectInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
	if insights.InsightsType == Direct {

		_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
		if apiErr != nil {
			return nil, "", apiErr
		}

		source := fmt.Sprintf("insights:direct:%s", resource.Service)

		var data DirectInsightsData
		fmt.Print("About to unmarshall the following:")
		fmt.Println(v)

		err := json.Unmarshal([]byte(v), &data.Data)
		if err != nil {
			return nil, "", err
		}
		fmt.Println(environment)
		fmt.Println(source)
		fmt.Println(data)
		for k, v := range data.Data {
			fmt.Printf("%v:%v\n", k, v.Description)
		}
		//
		//facts, err := processFactsFromDirect(DirectInsightsData, environment.Id, source)
		//if err != nil {
		//	return nil, "", err
		//}

		//facts, err = KeyFactsFilter(facts)
		//if err != nil {
		//	return nil, "", err
		//}
		//
		//return facts, source, nil
	}
	return []LagoonFact{}, "", nil
}

//func processFactsFromDirect(imageInspectData DirectInsightsData, id int, source string) ([]LagoonFact, error) {
//	var factsInput []LagoonFact
//
//	var filteredFacts []EnvironmentVariable
//	keyFactsExistMap := make(map[string]bool)
//
//	// Check if image inspect contains useful environment variables
//	if imageInspectData.Env != nil {
//		for _, v := range imageInspectData.Env {
//			var envSplitStr = strings.Split(v, "=")
//			env := EnvironmentVariable{
//				Key:   envSplitStr[0],
//				Value: envSplitStr[1],
//			}
//
//			// Remove duplicate key facts
//			if _, ok := keyFactsExistMap[env.Key]; !ok {
//				keyFactsExistMap[env.Key] = true
//				filteredFacts = append(filteredFacts, env)
//			}
//		}
//	}
//
//	for _, f := range filteredFacts {
//
//		fact := LagoonFact{
//			Environment: id,
//			Name:        f.Key,
//			Value:       f.Value,
//			Source:      source,
//			Description: "Environment Variable",
//			KeyFact:     false,
//			Type:        FactTypeText,
//		}
//		fmt.Println("Processing fact name " + f.Key)
//		fact, _ = ProcessLagoonFactAgainstRegisteredFilters(fact, f)
//		factsInput = append(factsInput, fact)
//	}
//	return factsInput, nil
//}

func init() {
	RegisterParserFilter(processDirectInsightsData)
}
