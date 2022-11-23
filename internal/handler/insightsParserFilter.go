package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/Khan/genqlient/graphql"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func processSbomInsightsData(h *Messaging, insights InsightsData, v string, apiClient graphql.Client, resource ResourceDestination) ([]LagoonFact, string, error) {
	if insights.InsightsType == Sbom {
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
				panic(err)
			}
		}

		// Determine lagoon resource destination
		_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
		if apiErr != nil {
			return nil, "", apiErr
		}
		source := fmt.Sprintf("insights:sbom:%s", resource.Service)

		// Process SBOM into facts
		facts := processFactsFromSBOM(bom.Components, environment.Id, source, resource.Service)

		facts, err = KeyFactsFilter(facts)
		if err != nil {
			return nil, "", err
		}

		facts, err = FactDuplicateHandler(facts)
		if err != nil {
			return nil, "", err
		}

		if len(facts) == 0 {
			return nil, "", fmt.Errorf("no facts to process")
		}

		log.Printf("Successfully decoded SBOM of image %s\n", bom.Metadata.Component.Name)
		log.Printf("- Generated: %s with %s\n", bom.Metadata.Timestamp, (*bom.Metadata.Tools)[0].Name)
		log.Printf("- Packages found: %d\n", len(*bom.Components))

		return facts, source, nil
	}
	return []LagoonFact{}, "", nil
}

func processFactsFromSBOM(facts *[]cdx.Component, environmentId int, source string, service string) []LagoonFact {
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
			Service:     service,
			Description: f.PackageURL,
			KeyFact:     false,
			OriginalFact: PreTransformedFact{
				Name:  f.Name,
				Value: f.Version,
			},
			Type: FactTypeText,
		}
		fmt.Println("Processing fact name " + f.Name)
		fact, _ = ProcessLagoonFactAgainstRegisteredFilters(fact, f)
		factsInput = append(factsInput, fact)
	}
	return factsInput
}

func init() {
	RegisterParserFilter(processSbomInsightsData)
}
