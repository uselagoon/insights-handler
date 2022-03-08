package handler

import "github.com/Khan/genqlient/graphql"

var parserFilters []func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error)

func RegisterParserFilter(pf func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error)) {
	parserFilters = append(parserFilters, pf)
}
