package handler

import (
	"encoding/json"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/goccy/go-yaml"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"os"
	"reflect"
	"testing"
)

func Test_executeProcessing(t *testing.T) {
	type args struct {
		bomLocation string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test1",
			args: args{bomLocation: "./testassets/grypeExecuteProcessing_test1.json"},
		},
	}

	//Let's ensure that grype is available locally
	grypePath := "./testassets/bin/trivy"
	if _, err := os.Stat(grypePath); os.IsNotExist(err) {
		t.Errorf("Grype not found at %v - please run `make gettestgrype`", grypePath)
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bomText, _ := os.ReadFile(tt.args.bomLocation)
			var bom cyclonedx.BOM
			err := json.Unmarshal(bomText, &bom)
			got, err := executeProcessing(grypePath, bom)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeProcessing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//we're just testing that there are vulnerabilities
			if len(*got.Vulnerabilities) == 0 {
				t.Errorf("Grype integration seems to be failing")
			}
		})
	}
}

func Test_convertBOMToProblemsArray(t *testing.T) {
	type args struct {
		environment int
		source      string
		service     string
		bomLocation string
	}
	tests := []struct {
		name             string
		args             args
		numberOfProblems int
		wantErr          bool
	}{
		{
			name: "test1",
			args: args{
				environment: 0,
				source:      "test1",
				service:     "cli",
				bomLocation: "./testassets/bomToProblems_test1.json",
			},
			numberOfProblems: 191,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bomText, _ := os.ReadFile(tt.args.bomLocation)
			var bom cyclonedx.BOM
			json.Unmarshal(bomText, &bom)
			got, err := convertBOMToProblemsArray(tt.args.environment, tt.args.source, tt.args.service, bom)
			fmt.Print(len(got))
			if (err != nil) != tt.wantErr {
				t.Errorf("convertBOMToProblemsArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.numberOfProblems {
				t.Errorf("convertBOMToProblemsArray() got #problems %v, expected %v #problems", got, tt.numberOfProblems)
			}
		})
	}
}

func Test_trivyReportToProblems(t *testing.T) {
	type args struct {
		environment  int
		source       string
		service      string
		reportOnDisk string
	}
	tests := []struct {
		name    string
		args    args
		want    []lagoonclient.LagoonProblem
		wantErr bool
	}{
		{
			name: "testing loading from disk",
			args: args{
				environment:  0,
				source:       "testsource",
				service:      "testservice",
				reportOnDisk: "./testassets/trivySbomScanResponse.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//let's load a report from disk

			fileContents, err := os.ReadFile(tt.args.reportOnDisk)
			var report types.Report

			if err != nil {
				t.Errorf(err.Error())
				return
			}

			err = yaml.Unmarshal(fileContents, &report)
			if err != nil {
				t.Errorf(err.Error())
				return
			}

			got, err := trivyReportToProblems(tt.args.environment, tt.args.source, tt.args.service, report)
			if (err != nil) != tt.wantErr {
				t.Errorf("trivyReportToProblems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trivyReportToProblems() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_executeProcessingTrivy(t *testing.T) {
	type args struct {
		trivyRemoteAddress string
		bomWriteDir        string
		bomFile            string
	}
	tests := []struct {
		name              string
		args              args
		numberProblemsMin int
		wantErr           bool
	}{
		{
			name:              "Basic test",
			numberProblemsMin: 20,
			args: args{
				trivyRemoteAddress: "http://localhost:4954",
				bomWriteDir:        "/tmp/",
				bomFile:            "testassets/bomToProblems_test1.json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// load up the bom
			fileData, err := os.ReadFile(tt.args.bomFile)
			if err != nil {
				t.Errorf("Unable to open sbom file '%v' error '%v'", tt.args.bomFile, err.Error())
				return
			}

			var testBom cyclonedx.BOM

			err = json.Unmarshal(fileData, &testBom)
			if err != nil {
				t.Errorf("Unable to parse sbom file '%v' error %v ", tt.args.bomFile, err.Error())
				return
			}

			got, err := executeProcessingTrivy(tt.args.trivyRemoteAddress, tt.args.bomWriteDir, testBom)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeProcessingTrivy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			problems, err := trivyReportToProblems(0, "", "", got)
			if err != nil {
				t.Errorf("%v", err.Error())
				return
			}

			if len(problems) < tt.numberProblemsMin {
				t.Errorf("Number of problems inaccurate got %v, wanted more than %v", len(problems), tt.numberProblemsMin)
			}
		})
	}
}
