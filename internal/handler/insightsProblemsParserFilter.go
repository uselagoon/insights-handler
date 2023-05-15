package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Khan/genqlient/graphql"
)

type ProblemsPayload struct {
	Problems []LagoonProblem `json:"problems,omitempty"`
}

type DirectProblemsInsightsData struct {
	Data map[string]LagoonProblem `json:"data,omitempty"`
}

func processProblemsInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonProblem, string, error) {
	source := fmt.Sprintf("insights:problems:%s", insights.InputType)

	if insights.LagoonType == Problems && insights.InsightsType == Raw {
		r := strings.NewReader(v)

		res, err := ioutil.ReadAll(r)
		if err != nil {
			fmt.Println("err: ", err)
		}

		problems := processProblemsFromJSON(res, source)
		if len(problems) == 0 {
			return nil, "", fmt.Errorf("no problems to process")
		}

		log.Printf("Successfully processed problems")
		log.Printf("- problems found: %d\n", len(problems))

		return problems, source, nil
	}

	if insights.InsightsType == Direct {

		data := DirectProblemsInsightsData{}
		err := json.Unmarshal([]byte(v), &data)
		if err != nil {
			fmt.Println("err: ", err)
			return nil, "", err
		}

		problems := []LagoonProblem{}
		for _, problem := range data.Data {
			problems = append(problems, problem)
		}

		log.Printf("Successfully processed problems")
		log.Printf("- problems found: %d\n", len(problems))

		return problems, source, nil
	}

	return nil, "", nil
}

func processProblemsFromJSON(problems []byte, source string) []LagoonProblem {
	var problemsInput []LagoonProblem

	var problemsPayload ProblemsPayload
	err := json.Unmarshal(problems, &problemsPayload)
	if err != nil {
		fmt.Println(err)
		panic("Can't unmarshal problems")
	}

	if len(problemsPayload.Problems) == 0 {
		return problemsInput
	}

	for _, p := range problemsPayload.Problems {
		problem := LagoonProblem{
			Identifier:        p.Identifier,
			Environment:       p.Environment,
			AssociatedPackage: p.AssociatedPackage,
			Version:           p.Version,
			FixedVersion:      p.FixedVersion,
			Source:            p.Source,
			Service:           p.Service,
			Data:              p.Data,
			Severity:          p.Severity,
			SeverityScore:     p.SeverityScore,
			Description:       p.Description,
			Links:             p.Links,
		}
		fmt.Println("Processing problem: " + p.Identifier)
		problemsInput = append(problemsInput, problem)
	}
	return problemsInput
}

func init() {
	RegisterParserFilter(processProblemsInsightsData)
}
