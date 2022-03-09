package handler

func KeyFactsFilter(factsInput []LagoonFact) ([]LagoonFact, error) {
	var filteredFacts []LagoonFact
	for _, v := range factsInput {
		if v.KeyFact {
			filteredFacts = append(filteredFacts, v)
		}
	}
	return filteredFacts, nil
}
