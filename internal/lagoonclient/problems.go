package lagoonclient

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
)

func AddProblems(ctx context.Context, client graphql.Client, problems []AddProblemInput) (string, error) {
	resp, err := addProblems(ctx, client, problems)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Added %d problems", len(resp.AddProblems)), nil
}

func DeleteProblemsFromSource(ctx context.Context, client graphql.Client, environmentID int, service string, source string) (string, error) {
	resp, err := deleteProblemsFromSource(ctx, client, environmentID, service, source)
	if err != nil {
		return "", err
	}

	return resp.DeleteProblemsFromSource, nil
}
