package models

type Fact struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Environment string `json:"environment"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Category    string `json:"category"`
	KeyFact     bool   `json:"keyFact"`
	Type        string `json:"type"`
}
