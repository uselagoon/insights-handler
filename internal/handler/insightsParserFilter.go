package handler

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/Khan/genqlient/graphql"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"log/slog"
)

func processSbomInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, []lagoonclient.LagoonProblem, string, error) {

	source := fmt.Sprintf("insights:sbom:%s", resource.Service)
	logger := slog.With("ProjectName", resource.Project, "EnvironmentName", resource.Environment, "Source", source)

	// ret values
	var problemSlice []lagoonclient.LagoonProblem

	if insights.InsightsType != Sbom {
		return []LagoonFact{}, problemSlice, "", nil
	}

	bom, err := getBOMfromPayload(v)
	if err != nil {
		return []LagoonFact{}, problemSlice, "", err
	}

	// Determine lagoon resource destination
	_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		return nil, problemSlice, "", apiErr
	}

	// we process the SBOM here
	if h.ProblemsFromSBOM == true {
		isAlive, err := IsTrivyServerIsAlive(h.TrivyServerEndpoint)
		if err != nil {
			return nil, problemSlice, "", fmt.Errorf("trivy server not alive: %v", err.Error())
		} else {
			logger.Debug("Trivy is reachable")
		}
		if isAlive {
			problemSlice, err = SbomToProblems(apiClient, h.TrivyServerEndpoint, "/tmp/", environment.Id, resource.Service, *bom)
		}
		if err != nil {
			return nil, problemSlice, "", err
		}
	}

	// Process SBOM into facts
	facts := processFactsFromSBOM(logger, bom.Components, environment.Id, source)

	// Here, before we filter things, we run our facts through EOL data
	problemSlice = append(problemSlice, lagoonclient.LagoonProblem{
		Id:                0,
		Environment:       environment.Id,
		Identifier:        "Testing EOL",
		Version:           "1.1",
		FixedVersion:      "1.2",
		Source:            "",
		Service:           "",
		Data:              "",
		Severity:          "",
		SeverityScore:     0,
		AssociatedPackage: "",
		Description:       "",
		Links:             "",
	})

	facts, err = KeyFactsFilter(facts)
	if err != nil {
		return nil, problemSlice, "", err
	}

	if len(facts) == 0 {
		return nil, problemSlice, "", fmt.Errorf("no facts to process")
	}

	//log.Printf("Successfully decoded SBOM of image %s with %s, found %d for '%s:%s'", bom.Metadata.Component.Name, (*bom.Metadata.Tools)[0].Name, len(*bom.Components), resource.Project, resource.Environment)
	logger.Info("Successfully decoded SBOM",
		"image", bom.Metadata.Component.Name,
		"fieldName", (*bom.Metadata.Tools.Components)[0].Name,
		"Length", len(*bom.Components),
	)

	return facts, problemSlice, source, nil
}

func processFactsFromSBOM(logger *slog.Logger, facts *[]cdx.Component, environmentId int, source string) []LagoonFact {
	var factsInput []LagoonFact
	if facts == nil || len(*facts) == 0 {
		return factsInput
	}

	var filteredFacts []cdx.Component
	keyFactsExistMap := make(map[string]bool)

	// Filter key facts
	for _, v := range *facts {
		if _, ok := keyFactsExistMap[v.Name]; !ok {
			keyFactsExistMap[v.Name] = true
			filteredFacts = append(filteredFacts, v)
		}
	}

	for _, f := range filteredFacts {
		fact := LagoonFact{
			Environment: environmentId,
			Name:        f.Name,
			Value:       f.Version,
			Source:      source,
			Description: f.PackageURL,
			KeyFact:     false,
			Type:        FactTypeText,
		}
		//if EnableDebug {
		//	log.Println("[DEBUG] processing fact name " + f.Name)
		//}
		logger.Debug("Processing fact",
			"Name", f.Name,
			"Value", f.Version,
		)
		fact, _ = ProcessLagoonFactAgainstRegisteredFilters(fact, f)
		factsInput = append(factsInput, fact)
	}
	return factsInput
}
