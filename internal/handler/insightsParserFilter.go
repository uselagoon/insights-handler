package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/Khan/genqlient/graphql"
)

func processSbomInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {

	source := fmt.Sprintf("insights:sbom:%s", resource.Service)
	logger := slog.With("ProjectName", resource.Project, "EnvironmentName", resource.Environment, "Source", source)

	if insights.InsightsType != Sbom {
		return []LagoonFact{}, "", nil
	}

	bom := new(cdx.BOM)

	// Decode base64
	r := strings.NewReader(v)
	dec := base64.NewDecoder(base64.StdEncoding, r)

	res, err := ioutil.ReadAll(dec)
	if err != nil {
		return nil, "", err
	}

	fileType := http.DetectContentType(res)

	if fileType != "application/zip" && fileType != "application/x-gzip" && fileType != "application/gzip" {
		decoder := cdx.NewBOMDecoder(bytes.NewReader(res), cdx.BOMFileFormatJSON)
		if err = decoder.Decode(bom); err != nil {
			return nil, "", err
		}
	} else {
		// Compressed cyclonedx sbom
		result, decErr := decodeGzipString(v)
		if decErr != nil {
			return nil, "", decErr
		}
		b, mErr := json.MarshalIndent(result, "", " ")
		if mErr != nil {
			return nil, "", mErr
		}

		decoder := cdx.NewBOMDecoder(bytes.NewReader(b), cdx.BOMFileFormatJSON)
		if err = decoder.Decode(bom); err != nil {
			return nil, "", err
		}
	}

	// Determine lagoon resource destination
	_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		return nil, "", apiErr
	}

	// we process the SBOM here

	if h.ProblemsFromSBOM == true {
		isAlive, err := IsTrivyServerIsAlive(h.TrivyServerEndpoint)
		if err != nil {
			return nil, "", fmt.Errorf("trivy server not alive: %v", err.Error())
		} else {
			logger.Debug("Trivy is reachable")
		}
		if isAlive {
			err = SbomToProblems(apiClient, h.TrivyServerEndpoint, "/tmp/", environment.Id, resource.Service, *bom)
		}
		if err != nil {
			return nil, "", err
		}
	}

	// Process SBOM into facts
	facts := processFactsFromSBOM(logger, bom.Components, environment.Id, source)

	facts, err = KeyFactsFilter(facts)
	if err != nil {
		return nil, "", err
	}

	if len(facts) == 0 {
		return nil, "", fmt.Errorf("no facts to process")
	}

	//log.Printf("Successfully decoded SBOM of image %s with %s, found %d for '%s:%s'", bom.Metadata.Component.Name, (*bom.Metadata.Tools)[0].Name, len(*bom.Components), resource.Project, resource.Environment)
	logger.Info("Successfully decoded SBOM",
		"image", bom.Metadata.Component.Name,
		"fieldName", (*bom.Metadata.Tools)[0].Name,
		"Length", len(*bom.Components),
	)

	return facts, source, nil
}

func processFactsFromSBOM(logger *slog.Logger, facts *[]cdx.Component, environmentId int, source string) []LagoonFact {
	var factsInput []LagoonFact
	if len(*facts) == 0 {
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

func init() {
	RegisterParserFilter(processSbomInsightsData)
}
