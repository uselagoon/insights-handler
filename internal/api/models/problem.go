package models

type Problem struct {
	ID          int    `json:"id"`
	Identifier  string `json:"identifier"`
	Value       string `json:"value"`
	Environment string `json:"environment"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Category    string `json:"category"`
	KeyFact     bool   `json:"keyFact"`
}
