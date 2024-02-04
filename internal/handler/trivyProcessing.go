package handler

import (
	"context"
	"encoding/json"
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/Khan/genqlient/graphql"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const problemSource = "insights-handler-trivy"

func SbomToProblems(apiClient graphql.Client, trivyRemoteAddress string, bomWriteDirectory string, environmentId int, service string, sbom cdx.BOM) error {
	problemsArray, err := executeProcessingTrivy(trivyRemoteAddress, bomWriteDirectory, sbom)
	if err != nil {
		return fmt.Errorf("unable to execute trivy processing: %v", err.Error())
	}

	for i := 0; i < len(problemsArray); i++ {
		problemsArray[i].Environment = environmentId
		problemsArray[i].Service = service
		problemsArray[i].Source = problemSource
	}

	err = writeProblemsArrayToApi(apiClient, environmentId, problemSource, service, problemsArray)
	if err != nil {
		return fmt.Errorf("unable to execute trivy processing- writing problems to api: %v", err.Error())
	}
	return nil
}

func convertBOMToProblemsArray(environment int, source string, service string, bom cdx.BOM) ([]lagoonclient.LagoonProblem, error) {
	var ret []lagoonclient.LagoonProblem
	if bom.Vulnerabilities == nil {
		return ret, fmt.Errorf("No Vulnerabilities")
	}
	vulnerabilities := *bom.Vulnerabilities
	for _, v := range vulnerabilities {

		p := lagoonclient.LagoonProblem{
			Environment:       environment,
			Identifier:        v.ID,
			Version:           "",
			FixedVersion:      "",
			Source:            source,
			Service:           service,
			Data:              "{}",
			AssociatedPackage: "",
			Description:       v.Description,
			Links:             v.Source.URL,
		}
		if v.Affects != nil && len(*v.Affects) > 0 {
			p.AssociatedPackage = (*v.Affects)[0].Ref //v.Affects[0].Ref
		}
		//here we need to ensure that there are actually vulnerabilities
		if v.Ratings != nil && len(*v.Ratings) > 0 {

			p.Severity = lagoonclient.ProblemSeverityRating(strings.ToUpper(string((*v.Ratings)[0].Severity)))
			var sevScore float64

			if (*v.Ratings)[0].Score != nil {
				sevScore = *(*v.Ratings)[0].Score
			}
			if sevScore > 1 {
				sevScore = sevScore / 10
			}
			p.SeverityScore = sevScore
		}
		ret = append(ret, p)
	}
	return ret, nil
}

func writeProblemsArrayToApi(apiClient graphql.Client, environment int, source string, service string, problems []lagoonclient.LagoonProblem) error {

	ret, err := lagoonclient.DeleteProblemsFromSource(context.TODO(), apiClient, environment, service, source)
	if err != nil {
		return err
	}
	//fmt.Printf("Deleted problems from API for %v:%v - response: %v\n", service, source, ret)
	slog.Info("Deleted problems from API",
		"Service", service,
		"Source", source,
		"Return Data", ret,
	)

	//now we write the problems themselves
	_, err = lagoonclient.AddProblems(context.TODO(), apiClient, problems)

	if err != nil {
		return err
	}

	return nil
}

func IsTrivyServerIsAlive(trivyRemoteAddress string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("%v/healthz", trivyRemoteAddress))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	body := string(bodyBytes)

	return body == "ok", nil
}

type trivyOutput struct {
	Results []trivyOutputResults `json:"Results"`
}

type trivyOutputResults struct {
	Vulnerabilities []TrivyProblemOutput `json:"Vulnerabilities"`
}

type TrivyProblemOutput struct {
	VulnerabilityID  string   `json:"VulnerabilityID,omitempty"`
	PkgID            string   `json:"PkgID,omitempty"`
	PkgName          string   `json:"PkgName,omitempty"`
	InstalledVersion string   `json:"InstalledVersion,omitempty"`
	FixedVersion     string   `json:"FixedVersion,omitempty"`
	Status           string   `json:"Status,omitempty"`
	SeveritySource   string   `json:"SeveritySource,omitempty"`
	PrimaryURL       string   `json:"PrimaryURL,omitempty"`
	PkgRef           string   `json:"PkgRef,omitempty"`
	Title            string   `json:"Title,omitempty"`
	Description      string   `json:"Description,omitempty"`
	Severity         string   `json:"Severity,omitempty"`
	CweIDs           []string `json:"CweIDs,omitempty"`
	References       []string `json:"References,omitempty"`
	PublishedDate    string   `json:"PublishedDate,omitempty"`
	LastModifiedDate string   `json:"LastModifiedDate,omitempty"`
}

func executeProcessingTrivy(trivyRemoteAddress string, bomWriteDir string, bom cdx.BOM) ([]lagoonclient.LagoonProblem, error) {

	//first, we write this thing to disk
	slog.Info("About to process trivy details locally")

	file, err := os.CreateTemp(bomWriteDir, "cycloneDX-*.json")
	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	marshalledBom, err := json.Marshal(bom)

	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	_, err = file.Write(marshalledBom)
	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	fullFilename := fmt.Sprintf("%v/%v", bomWriteDir, fileInfo.Name())

	// Let's defer removing our file till the function returns
	defer func() {
		os.Remove(fullFilename)
		file.Close()
	}()

	cmd := exec.Command("trivy", "sbom", "--format", "json", "--server", trivyRemoteAddress, fullFilename)
	var out strings.Builder
	var outErr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &outErr

	err = cmd.Run()
	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	var results trivyOutput

	err = json.Unmarshal([]byte(out.String()), &results)
	if err != nil {
		return []lagoonclient.LagoonProblem{}, err
	}

	retData := []lagoonclient.LagoonProblem{}

	for _, resultset := range results.Results {
		for _, vul := range resultset.Vulnerabilities {
			prob := lagoonclient.LagoonProblem{
				Identifier:        vul.VulnerabilityID,
				Version:           vul.InstalledVersion,
				FixedVersion:      vul.FixedVersion,
				Data:              "{}",
				Severity:          lagoonclient.ProblemSeverityRating(vul.Severity),
				SeverityScore:     0,
				AssociatedPackage: vul.PkgName,
				Description:       vul.Description,
				Links:             "",
			}
			retData = append(retData, prob)
		}
	}

	return retData, nil
}
