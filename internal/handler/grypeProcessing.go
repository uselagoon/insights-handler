package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"io"
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

func processQueue() {
	for {
		i := sbomQueuePop()
		if i != nil {
			vulnerabilitiesBom, err := executeProcessing(queue.GrypeLocation, i.SBOM)
			if err != nil {
				fmt.Println("Unable to process queue item")
				//fmt.Println(i)
				//fmt.Print(err)
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
