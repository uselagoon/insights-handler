package handler

import (
	"context"
	"encoding/json"
	"github.com/Khan/genqlient/graphql"
	"github.com/cheshir/go-mq"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/service"
	"log/slog"
	"net/http"
	"strconv"
)

// processing.go contains the functions that actually process the incoming messages

func processFactsDirectly(message mq.Message, h *Messaging) string {
	var directFacts DirectFacts

	json.Unmarshal(message.Body(), &directFacts)
	err := json.Unmarshal(message.Body(), &directFacts)
	if err != nil {
		slog.Error("Could not unmarshal data", "Error", err.Error())
		return "exciting, unable to process direct facts"
	}

	// since it's useful to allow int and string json definitions, we need to convert strings here to ints.
	environmentId, err := strconv.Atoi(directFacts.EnvironmentId.String())
	if err != nil {
		slog.Error("Error converting EnvironmentId to int", "Error", err)
		return "exciting, unable to process direct facts"
	}

	slog.Debug("Facts info", "data", directFacts)

	apiClient := graphql.NewClient(h.LagoonAPI.Endpoint, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, h: h}})

	factSources := map[string]string{}

	processedFacts := make([]lagoonclient.AddFactInput, len(directFacts.Facts))
	for i, fact := range directFacts.Facts {

		vartypeString := FactTypeText
		if fact.Type == FactTypeText || fact.Type == FactTypeSemver || fact.Type == FactTypeUrl {
			vartypeString = fact.Type
		}

		processedFacts[i] = lagoonclient.AddFactInput{
			Environment: environmentId,
			Name:        fact.Name,
			Value:       fact.Value,
			Source:      fact.Source,
			Description: fact.Description,
			KeyFact:     false,
			Type:        lagoonclient.FactType(vartypeString),
			Category:    fact.Category,
		}
		factSources[fact.Source] = fact.Source
	}

	for _, s := range factSources {
		_, err = lagoonclient.DeleteFactsFromSource(context.TODO(), apiClient, environmentId, s)
		if err != nil {

			slog.Error("Error deleting facts from source",
				"EnvironmentId", directFacts.EnvironmentId,
				"ProjectName", directFacts.ProjectName,
				"EnvironmentName", directFacts.EnvironmentName,
				"Source", s,
				"Error", err,
			)
		}

		// Now we do the same in the DB
		if h.DBConnection != nil {
			n, err := directFacts.EnvironmentId.Int64()
			if err != nil {
				slog.Error("Unable to convert json.Number to int64",
					"EnvironmentId", directFacts.EnvironmentId,
					"ProjectName", directFacts.ProjectName,
					"EnvironmentName", directFacts.EnvironmentName,
					"Source", s,
					"Error", err,
				)
			}
			if _, err := service.DeleteFacts(h.DBConnection, int(n), s); err != nil {
				slog.Error("Unable to delete facts from DB",
					"EnvironmentId", directFacts.EnvironmentId,
					"ProjectName", directFacts.ProjectName,
					"EnvironmentName", directFacts.EnvironmentName,
					"Source", s,
					"Error", err,
				)
			}

		}

		slog.Info("Deleted facts",
			"EnvironmentId", directFacts.EnvironmentId,
			"ProjectName", directFacts.ProjectName,
			"EnvironmentName", directFacts.EnvironmentName,
			"Source", s,
		)
	}

	facts, err := lagoonclient.AddFacts(context.TODO(), apiClient, processedFacts)
	if err != nil {
		//log.Println(err)
		slog.Error("Issue adding facts to API", "Error", err.Error())
	}

	// Now add facts to DB
	if h.DBConnection != nil {
		lagoonFacts := []lagoonclient.Fact{}
		for _, e := range processedFacts {
			lf := lagoonclient.Fact{
				Environment: environmentId,
				Name:        e.Name,
				Value:       e.Value,
				Source:      e.Source,
				Description: e.Description,
				KeyFact:     e.KeyFact,
				Type:        e.Type,
				Category:    e.Category,
			}
			lagoonFacts = append(lagoonFacts, lf)
		}

		err = service.CreateFacts(h.DBConnection, &lagoonFacts)
		if err != nil {
			slog.Error("Issue adding facts to DB", "Error", err.Error())
		}
	}

	return facts
}

func processProblemsDirectly(message mq.Message, h *Messaging) ([]string, error) {
	var directProblems DirectProblems
	json.Unmarshal(message.Body(), &directProblems)
	err := json.Unmarshal(message.Body(), &directProblems)
	if err != nil {
		slog.Error("Could not unmarshal JSON", "Error", err)
		return []string{}, err
	}

	slog.Debug("Problems data", "data", directProblems)

	apiClient := graphql.NewClient(h.LagoonAPI.Endpoint, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, h: h}})

	// serviceSource just gives us simple structure to do the deletions
	type serviceSource struct {
		Source  string
		Service string
	}
	problemSources := map[string]serviceSource{}

	for i, problem := range directProblems.Problems {

		// We want to ensure that the incoming problems aren't malformed or trying to do anything dodgy with env ids

		if problem.Environment != directProblems.EnvironmentId {
			directProblems.Problems[i].Environment = directProblems.EnvironmentId
		}

		problemSources[problem.Service+problem.Source] = serviceSource{
			Source:  problem.Source,
			Service: problem.Service,
		}
	}

	for _, s := range problemSources {
		_, err := lagoonclient.DeleteProblemsFromSource(context.TODO(), apiClient, directProblems.EnvironmentId, s.Service, s.Source)
		if err != nil {
			return []string{}, err
		}

		slog.Info("Deleted problems",
			"EnvironmentId", directProblems.EnvironmentId,
			"ProjectName", directProblems.ProjectName,
			"EnvironmentName", directProblems.EnvironmentName,
			"Source", s,
		)
	}

	resptext, err := lagoonclient.AddProblems(context.TODO(), apiClient, directProblems.Problems)
	if err != nil {
		return []string{}, err
	}

	return resptext, nil
}
