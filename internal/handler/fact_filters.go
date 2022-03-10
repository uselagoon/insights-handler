package handler

import "fmt"

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

	var factOccurenceTracker = map[string]int32{}

	var filteredFacts []LagoonFact
	for _, v := range factsInput {

		if val, ok := factOccurenceTracker[v.Name]; ok {
			factOccurenceTracker[v.Name] += 1
			v.Name = fmt.Sprintf("%v [%v]", v.Name, val)
		} else {
			factOccurenceTracker[v.Name] = 1
		}

		filteredFacts = append(filteredFacts, v)

	}
	return filteredFacts, nil
}
