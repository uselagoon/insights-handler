package handler

import (
	"github.com/Khan/genqlient/graphql"
)

type parserFilter interface {
	isFilteredOut() bool
	hasError() bool
	getError() error
	isOfType(typename string) parserFilter
	fieldContains(fieldname string, regex string) parserFilter
	fieldContainsExactMatch(fieldname string, match string) parserFilter
	setKeyFact(isKeyFact bool) parserFilter
	setFactField(fieldname string, value string) parserFilter
	getFact() LagoonFact
}

var parserFilters []func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error)

func RegisterParserFilter(pf func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error)) {
	parserFilters = append(parserFilters, pf)
}
