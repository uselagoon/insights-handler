package handler

import (
	"fmt"
	"regexp"
)

func KeyFactsFilter(factsInput []LagoonFact) ([]LagoonFact, error) {
	filteredFacts := make(map[string]LagoonFact)

	factRegexes, err := scanKeyFactsFile("./key_facts.txt")
	if err != nil {
		fmt.Errorf("scan file error: %w", err)
	}

	for _, v := range factsInput {
		for _, k := range factRegexes {
			hasMatch, err := regexp.Match(k, []byte(v.Name))
			if err != nil {
				fmt.Errorf(err.Error())
			}
			if hasMatch {
				if _, ok := filteredFacts[v.Name]; !ok {
					filteredFacts[v.Name] = v
				}
			}
		}
	}
	v := make([]LagoonFact, 0, len(filteredFacts))

	for _, value := range filteredFacts {
		v = append(v, value)
	}
	return v, nil
}
