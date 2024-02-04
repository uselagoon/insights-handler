package handler

import (
	"encoding/json"
	"github.com/CycloneDX/cyclonedx-go"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"os"
	"testing"
)

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

func Test_executeProcessingTrivyLocally(t *testing.T) {
	type args struct {
		trivyRemoteAddress string
		bomWriteDir        string
		bomFile            string
	}
	tests := []struct {
		name              string
		args              args
		want              []lagoonclient.LagoonProblem
		numberProblemsMin int
		wantErr           bool
	}{
		{
			name:              "testing basic",
			numberProblemsMin: 20,
			args: args{
				trivyRemoteAddress: "",
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

			problems, err := executeProcessingTrivy(tt.args.trivyRemoteAddress, tt.args.bomWriteDir, testBom)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeProcessingTrivy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(problems) < tt.numberProblemsMin {
				t.Errorf("Number of problems inaccurate got %v, wanted more than %v", len(problems), tt.numberProblemsMin)
			}
		})
	}
}
