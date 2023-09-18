package handler

import (
	"context"
	"encoding/json"
	"github.com/Khan/genqlient/graphql"
	"github.com/cheshir/go-mq"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"log"
	"net/http"
	"strconv"
)

// processing.go contains the functions that actually process the incoming messages

func processItemsDirectly(message mq.Message, h *Messaging) string {
	var directFacts DirectFacts
	json.Unmarshal(message.Body(), &directFacts)
	err := json.Unmarshal(message.Body(), &directFacts)
	if err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return "exciting, unable to process direct facts"
	}

	// since its useful to allow int and string json definitions, we need to convert strings here to ints.
	environmentId, err := strconv.Atoi(directFacts.EnvironmentId.String())
	if err != nil {
		log.Println("Error converting EnvironmentId to int:", err)
		return "exciting, unable to process direct facts"
	}

	if h.EnableDebug {
		log.Print("[DEBUG] facts", directFacts)
	}

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
			log.Println(err)
		}
		log.Printf("Deleted facts on '%v:%v' for source %v", directFacts.ProjectName, directFacts.EnvironmentName, s)
	}

	facts, err := lagoonclient.AddFacts(context.TODO(), apiClient, processedFacts)
	if err != nil {
		log.Println(err)
	}

	return facts
}
