package lagoonclient

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
)

type LagoonProblem struct {
	Id                int                   `json:"id"`
	Environment       int                   `json:"environment"`
	Identifier        string                `json:"identifier"`
	Version           string                `json:"version,omitempty"`
	FixedVersion      string                `json:"fixedVersion,omitempty"`
	Source            string                `json:"source,omitempty"`
	Service           string                `json:"service,omitempty"`
	Data              string                `json:"data"`
	Severity          ProblemSeverityRating `json:"severity,omitempty"`
	SeverityScore     float64               `json:"severityScore,omitempty"`
	AssociatedPackage string                `json:"associatedPackage,omitempty"`
	Description       string                `json:"description,omitempty"`
	Links             string                `json:"links,omitempty"`
}

//const (
//	ProblemSeverityRatingNone       string = "NONE"
//	ProblemSeverityRatingUnknown    string = "UNKNOWN"
//	ProblemSeverityRatingNegligible string = "NEGLIGIBLE"
//	ProblemSeverityRatingLow        string = "LOW"
//	ProblemSeverityRatingMedium     string = "MEDIUM"
//	ProblemSeverityRatingHigh       string = "HIGH"
//	ProblemSeverityRatingCritical   string = "CRITICAL"
//)

func AddProblems(ctx context.Context, client graphql.Client, problems []LagoonProblem) ([]string, error) {
	var respText []string

	for _, problem := range problems {

		resp, err := addProblem(ctx,
			client,
			problem.Environment,
			problem.Severity,
			problem.SeverityScore,
			problem.Identifier,
			problem.Service,
			problem.Source,
			problem.AssociatedPackage, problem.Description, problem.Links, problem.Version, problem.FixedVersion, problem.Data)

		if err != nil {
			//return respText, err
			respText = append(respText, fmt.Sprintf("Error adding %v with id in api: %v - %v", problem.Identifier, resp.AddProblem.Id, err))
		} else {
			respText = append(respText, fmt.Sprintf("Added %v with id in api: %v", problem.Identifier, resp.AddProblem.Id))
		}
	}

	return respText, nil
}

func DeleteProblemsFromSource(ctx context.Context, client graphql.Client, environmentID int, service string, source string) (string, error) {

	resp, err := deleteProblemsFromSource(ctx, client, environmentID, source, service)
	if err != nil {
		return "", err
	}

	return resp.DeleteProblemsFromSource, nil
}
