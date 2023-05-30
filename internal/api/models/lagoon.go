package models

type Lagoon struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
}

type Project struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Lagoon Lagoon `json:"lagoon"`
}

type Environment struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
