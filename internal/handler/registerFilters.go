package handler

import "fmt"

var KeyFactFilters []func(filter parserFilter) parserFilter

func ProcessLagoonFactAgainstRegisteredFilters(fact LagoonFact, insightsRawData interface{}) (LagoonFact, error) {
	for _, filter := range KeyFactFilters {
		pf := FactProcessor{
			Fact:          fact,
			InsightsData:  insightsRawData,
			hasErrorState: false,
			theError:      nil,
			filteredOut:   false,
		}
		pfout := filter(&pf)
		if !pfout.isFilteredOut() {
			fact = pfout.getFact()
		}
	}
	return fact, nil
}

func registerFilter(filter func(filter parserFilter) parserFilter) {
	KeyFactFilters = append(KeyFactFilters, filter)
}

func init() {

	// Let's register the standard key fact filters as they currently exist
	factRegexes, err := scanKeyFactsFile("./syft_key_facts.txt")
	if err != nil {
		fmt.Errorf("scan file error: %w", err)
	}
	for _, k := range factRegexes {
		workingRegex := k
		registerFilter(func(filter parserFilter) parserFilter {
			return filter.fieldContains("Name", workingRegex).setKeyFact(true)
		})
	}

	// Let's register the standard key fact filters as they currently exist
	factRegexes, err = scanKeyFactsFile("./env_var_key_facts.txt")
	if err != nil {
		fmt.Errorf("scan file error: %w", err)
	}
	for _, k := range factRegexes {
		workingRegex := k
		registerFilter(func(filter parserFilter) parserFilter {
			return filter.fieldContains("Key", workingRegex).setKeyFact(true)
		})
	}

}
