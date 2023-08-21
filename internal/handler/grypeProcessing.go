package handler

import (
	"encoding/json"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"io"
	"os/exec"
	"sync"
	"time"
)

type sbomQueueItem struct {
	EnvironmentId int
	SBOM          cyclonedx.BOM
}

type sbomQueue struct {
	Items         []sbomQueueItem
	Lock          sync.Mutex
	GrypeLocation string
}

var queue = sbomQueue{
	Items: []sbomQueueItem{},
	Lock:  sync.Mutex{},
}

func SetUpQueue(grypeLocation string) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()
	queue.GrypeLocation = grypeLocation
}

func sbomQueuePush(i sbomQueueItem) {
	queue.Lock.Lock()
	defer queue.Lock.Unlock()
	queue.Items = append(queue.Items, i)
}

func sbomQueuePop() *sbomQueueItem {
	if len(queue.Items) > 0 {
		queue.Lock.Lock()
		defer queue.Lock.Unlock()
		i := queue.Items[0]
		queue.Items = append(queue.Items[:1], queue.Items[2:]...)
		return &i
	}
	return nil
}

func processQueue() {
	for {
		i := sbomQueuePop()
		if i != nil {
			//executeProcessing(queue.GrypeLocation, i)
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
			Data:              "",
			AssociatedPackage: "",
			Description:       v.Description,
			Links:             v.Source.URL,
		}
		if v.Affects != nil && len(*v.Affects) > 0 {
			p.AssociatedPackage = (*v.Affects)[0].Ref //v.Affects[0].Ref
		}
		//here we need to ensure that there are actually vulnerabilities
		if v.Ratings != nil && len(*v.Ratings) > 0 {
			//Might make sense to grab the highest?
			//p.Severity = (*v.Ratings)[0].Severity

			//p.SeverityScore = *(*v.Ratings)[0].Score
		}
		ret = append(ret, p)
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

	fmt.Println("Output:", string(output))

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
