package handler

import (
	"context"
	"encoding/json"
	"github.com/Khan/genqlient/graphql"
	"github.com/cheshir/go-mq"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
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

	//if h.EnableDebug {
	//	log.Print("[DEBUG] facts", directFacts)
	//}
	//
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
			Service:     fact.Service,
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
		//log.Printf("Deleted facts on '%v:%v' for source %v\n", directFacts.ProjectName, directFacts.EnvironmentName, s)
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
		slog.Error("Issue adding facts", "Error", err.Error())
	}

	return facts
}

func deleteProblemsDirectly(message mq.Message, h *Messaging) (string, error) {
	var deleteMessage DirectDeleteMessage
	err := json.Unmarshal(message.Body(), &deleteMessage)
	if err != nil {
		return "", err
	}
	ret, err := lagoonclient.DeleteProblemsFromSource(context.TODO(), h.getApiClient(), deleteMessage.EnvironmentId, deleteMessage.Service, deleteMessage.Source)
	if err != nil {
		slog.Error("Unable to delete facts", "Error", err.Error(), "EnvironmentId", deleteMessage.EnvironmentId, "source", deleteMessage.Source, "service", deleteMessage.Service)
		return "", err
	}

	slog.Info("Deleted problems", "EnvironmentId", deleteMessage.EnvironmentId, "source", deleteMessage.Source, "service", deleteMessage.Service)

	return ret, nil
}

func deleteFactsDirectly(message mq.Message, h *Messaging) (string, error) {
	var deleteMessage DirectDeleteMessage
	err := json.Unmarshal(message.Body(), &deleteMessage)
	if err != nil {
		return "", err
	}
	ret, err := lagoonclient.DeleteFactsFromSource(context.TODO(), h.getApiClient(), deleteMessage.EnvironmentId, deleteMessage.Source)
	if err != nil {
		slog.Error("Unable to delete facts", "Error", err.Error(), "EnvironmentId", deleteMessage.EnvironmentId, "source", deleteMessage.Source, "service", deleteMessage.Service)
		return "", err
	}
	slog.Info("Deleted facts", "EnvironmentId", deleteMessage.EnvironmentId, "source", deleteMessage.Source, "service", deleteMessage.Service)

	return ret, nil
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
