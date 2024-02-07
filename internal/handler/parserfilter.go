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

type ParserFilterFunc func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error)

var parserFilters []ParserFilterFunc

func RegisterParserFilter(pf ParserFilterFunc) {
	parserFilters = append(parserFilters, func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
		result, source, err := pf(h, insights, v, apiClient, resource)
		if err != nil {
			return nil, "", err
		}
		return result, source, nil
	})
}
