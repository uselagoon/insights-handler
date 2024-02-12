package handler

// KeyFactsFilter simply takes a slice of LagoonFacts and filters out anything not marked with KeyFact = true
// This is used downstream to ensure only key facts are written to the DB
func KeyFactsFilter(factsInput []LagoonFact) ([]LagoonFact, error) {
	var filteredFacts []LagoonFact
	for _, v := range factsInput {
		if v.KeyFact {
			filteredFacts = append(filteredFacts, v)
		}
	}
	return filteredFacts, nil
}
