package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"github.com/aquasecurity/trivy/pkg/commands/artifact"
	"github.com/aquasecurity/trivy/pkg/flag"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const problemSource = "insights-handler-grype"

type sbomQueueItem struct {
	EnvironmentId int
	Service       string
	SBOM          cyclonedx.BOM
}

type sbomQueue struct {
	Items         []sbomQueueItem
	Lock          sync.Mutex
	GrypeLocation string
	Messaging     Messaging
}

var queue = sbomQueue{
	Items: []sbomQueueItem{},
	Lock:  sync.Mutex{},
}

func SetUpQueue(messageHandler Messaging, grypeLocation string) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()
	queue.GrypeLocation = grypeLocation
	queue.Messaging = messageHandler
}

func SbomQueuePush(i sbomQueueItem) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()
	queue.Items = append(queue.Items, i)
}

func sbomQueuePop() *sbomQueueItem {
	if len(queue.Items) > 0 {
		queue.Lock.Lock()
		defer queue.Lock.Unlock()
		i := queue.Items[0]
		queue.Items = queue.Items[1:]
		return &i
	}
	return nil
}

func SbomToProblems(trivyRemoteAddress string, bomWriteDirectory string, environmentId int, service string, sbom cyclonedx.BOM) error {
	rep, err := executeProcessingTrivy(trivyRemoteAddress, bomWriteDirectory, sbom)
	if err != nil {
		return err
	}

	problems, err := trivyReportToProblems(environmentId, problemSource, service, rep)

	if err != nil {
		return err
	}

	err = writeProblemsArrayToApi(environmentId, problemSource, service, problems)

	if err != nil {
		return err
	}

	return nil
}

func processQueue() {
	for {
		i := sbomQueuePop()
		if i != nil {
			vulnerabilitiesBom, err := executeProcessing(queue.GrypeLocation, i.SBOM)
			if err != nil {
				fmt.Println("Unable to process queue item")
				continue
			}
			problemArray, err := convertBOMToProblemsArray(i.EnvironmentId, problemSource, i.Service, vulnerabilitiesBom)
			if err != nil {
				fmt.Println("Unable to convert vulnerabilities list to problems array")
				//fmt.Println(vulnerabilitiesBom)
				fmt.Print(err)
				continue
			}
			err = writeProblemsArrayToApi(i.EnvironmentId, problemSource, i.Service, problemArray)
			if err != nil {
				fmt.Println("Unable to write problemArray to API")
				//fmt.Println(problemArray)
				fmt.Print(err)
				continue
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

func convertBOMToProblemsArray(environment int, source string, service string, bom cyclonedx.BOM) ([]lagoonclient.LagoonProblem, error) {
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

			//TODO: this is gross, fix it.
			p.Severity = lagoonclient.ProblemSeverityRating(strings.ToUpper(string((*v.Ratings)[0].Severity)))

			sevScore := *(*v.Ratings)[0].Score

			if sevScore > 1 {
				sevScore = sevScore / 10
			}

			p.SeverityScore = sevScore //*(*v.Ratings)[0].Score
		}
		ret = append(ret, p)
	}
	return ret, nil
}

func writeProblemsArrayToApi(environment int, source string, service string, problems []lagoonclient.LagoonProblem) error {

	ret, err := lagoonclient.DeleteProblemsFromSource(context.TODO(), queue.Messaging.getApiClient(), environment, service, source)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted problems from API for %v:%v - response: %v", service, source, ret)

	//now we write the problems themselves
	_, err = lagoonclient.AddProblems(context.TODO(), queue.Messaging.getApiClient(), problems)

	if err != nil {
		return err
	}

	return nil
}

func testTrivyServerIsAlive(trivyRemoteAddress string) (bool, error) {
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

func executeProcessingTrivy(trivyRemoteAddress string, bomWriteDir string, bom cyclonedx.BOM) (types.Report, error) {
	//first, we write this thing to disk
	file, err := os.CreateTemp(bomWriteDir, "cycloneDX-*.json")
	if err != nil {
		return types.Report{}, err
	}

	marshalledBom, err := json.Marshal(bom)

	if err != nil {
		return types.Report{}, err
	}

	_, err = file.Write(marshalledBom)
	if err != nil {
		return types.Report{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return types.Report{}, err
	}

	fullFilename := fmt.Sprintf("%v/%v", bomWriteDir, fileInfo.Name())

	// Let's defer removing our file till the function returns
	defer func() {
		os.Remove(fullFilename)
		file.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
	defer cancel()

	opts := flag.Options{
		GlobalOptions: flag.GlobalOptions{
			ConfigFile: "trivy.yaml",
			CacheDir:   "/tmp/.cache/trivy",
		},
		AWSOptions: flag.AWSOptions{},
		CacheOptions: flag.CacheOptions{
			CacheBackend: "fs",
		},
		CloudOptions: flag.CloudOptions{},
		DBOptions: flag.DBOptions{
			DBRepository:     "ghcr.io/aquasecurity/trivy-db",
			JavaDBRepository: "ghcr.io/aquasecurity/trivy-java-db",
		},
		ImageOptions:    flag.ImageOptions{},
		K8sOptions:      flag.K8sOptions{},
		LicenseOptions:  flag.LicenseOptions{},
		MisconfOptions:  flag.MisconfOptions{},
		ModuleOptions:   flag.ModuleOptions{},
		RegistryOptions: flag.RegistryOptions{},
		RegoOptions:     flag.RegoOptions{},
		RemoteOptions: flag.RemoteOptions{
			ServerAddr:    trivyRemoteAddress,
			Token:         "",
			TokenHeader:   "Trivy-Token",
			CustomHeaders: http.Header{},
		},
		RepoOptions:   flag.RepoOptions{},
		ReportOptions: flag.ReportOptions{},
		SBOMOptions:   flag.SBOMOptions{},
		ScanOptions: flag.ScanOptions{
			Target: fullFilename,
			Scanners: types.Scanners{
				types.VulnerabilityScanner,
			},
		},
		SecretOptions: flag.SecretOptions{},
		VulnerabilityOptions: flag.VulnerabilityOptions{
			VulnType: []string{
				"os",
				"library",
			},
		},
		AppVersion:        "dev",
		DisabledAnalyzers: nil,
	}
	runner, err := artifact.NewRunner(ctx, opts)

	if err != nil {
		return types.Report{}, err
	}

	rep, err := runner.ScanSBOM(context.TODO(), opts)

	if err != nil {
		return types.Report{}, err
	}

	return rep, nil
}

func trivyReportToProblems(environment int, source string, service string, report types.Report) ([]lagoonclient.LagoonProblem, error) {
	var ret []lagoonclient.LagoonProblem
	if len(report.Results) == 0 {
		return ret, fmt.Errorf("No Vulnerabilities")
	}

	for _, res := range report.Results {
		for _, v := range res.Vulnerabilities {
			p := lagoonclient.LagoonProblem{
				Environment:       environment,
				Identifier:        v.VulnerabilityID,
				Version:           v.InstalledVersion,
				FixedVersion:      v.FixedVersion,
				Source:            source,
				Service:           service,
				Data:              "{}",
				AssociatedPackage: "",
				Description:       v.Vulnerability.Description,
				// Severity:
			}

			if len(v.Vulnerability.References) > 0 {
				p.Links = v.Vulnerability.References[0]
			}

			p.Severity = lagoonclient.ProblemSeverityRating(v.Vulnerability.Severity)

			ret = append(ret, p)
		}
	}
	return ret, nil
}

func executeProcessing(grypeLocation string, bom cyclonedx.BOM) (cyclonedx.BOM, error) {
	cmd := exec.Command(grypeLocation, "-o", "cyclonedx-json")
	// Set up pipes for stdin, stdout, and stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("Failed to create stdin pipe:", err)
		return cyclonedx.BOM{}, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Failed to create stdout pipe:", err)
		return cyclonedx.BOM{}, err
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Failed to create stderr pipe:", err)
		return cyclonedx.BOM{}, err
	}
	defer stderr.Close()

	sbomString, err := json.Marshal(bom)
	if err != nil {
		return cyclonedx.BOM{}, err
	}
	//let's push the sbom into the stdin
	if err := cmd.Start(); err != nil {
		fmt.Println("Failed to start command:", err)
		return cyclonedx.BOM{}, err
	}

	go func() {
		defer stdin.Close()
		_, err = io.WriteString(stdin, string(sbomString))
	}()

	if err != nil {
		fmt.Println("Could not write to grype", err)
		return cyclonedx.BOM{}, err
	}

	//execute
	// Read from stdout
	output := make([]byte, 0) // Buffer to store the output
	buf := make([]byte, 1024) // Read buffer
	for {
		n, err := stdout.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("Failed to read from stdout:", err)
			return cyclonedx.BOM{}, err
		}
		if n == 0 {
			break
		}
		output = append(output, buf[:n]...)
	}

	//fmt.Println("Output:", string(output))

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Println("Command execution failed:", err)
		return cyclonedx.BOM{}, err
	}

	var ret cyclonedx.BOM
	err = json.Unmarshal(output, &ret)
	if err != nil {
		fmt.Println("Unable to unmarshal data")
		return ret, err
	}

	return ret, nil
}
