package handler

import (
	"fmt"
)

func KeyFactsFilter(factsInput []LagoonFact) ([]LagoonFact, error) {
	var filteredFacts []LagoonFact
	for _, v := range factsInput {
		if v.KeyFact {
			filteredFacts = append(filteredFacts, v)
		}
	}
	return filteredFacts, nil
}

func FactDuplicateHandler(factsInput []LagoonFact) ([]LagoonFact, error) {
	var factOccurrenceTracker = map[string]int32{}

	var filteredFacts []LagoonFact
	for _, v := range factsInput {
		if _, ok := factOccurrenceTracker[v.Name]; ok {
			factOccurrenceTracker[v.Name] += 1
			v.Name = fmt.Sprintf("%v (%v)", v.Name, v.OriginalFact.Name)
		} else {
			factOccurrenceTracker[v.Name] = 1
		}
		filteredFacts = append(filteredFacts, v)
	}

	return filteredFacts, nil
}
