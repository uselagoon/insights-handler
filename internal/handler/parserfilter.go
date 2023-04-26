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

type ParserFilterFunc[T any] func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]T, string, error)

var parserFilters []ParserFilterFunc[interface{}]

// Since Go does not allow type conversions between slices of different types, ([]T and []interface{}) are considered different types, and you cannot assign one to the other.
// Therefore  this func will convert the []T slice to a []interface{} slice
func ToInterfaceSlice[T any](slice []T) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

func RegisterParserFilter[T any](pf ParserFilterFunc[T]) {
	parserFilters = append(parserFilters, func(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]interface{}, string, error) {
		facts, source, err := pf(h, insights, v, apiClient, resource)
		if err != nil {
			return nil, "", err
		}
		return ToInterfaceSlice(facts), source, nil
	})
}
