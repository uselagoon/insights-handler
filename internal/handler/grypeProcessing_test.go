package handler

import (
	"encoding/json"
	"fmt"
	"github.com/CycloneDX/cyclonedx-go"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func Test_executeProcessing(t *testing.T) {
	type args struct {
		grypeLocation string
		//bom           cyclonedx.BOM
		bomLocation string
	}
	tests := []struct {
		name    string
		args    args
		want    cyclonedx.BOM
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				grypeLocation: "/usr/local/bin/grype",
				//grypeLocation: "/usr/bin/cat",
				bomLocation: "./testassets/grypeExecuteProcessing_test1.json",
			},
		},
	}

	//Let's ensure that grype is available locally
	grypePath := "./testassets/bin/grype"
	if _, err := os.Stat(grypePath); os.IsNotExist(err) {
		t.Errorf("Grype not found at %v - please run `make gettestgrype`")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bomText, _ := os.ReadFile(tt.args.bomLocation)
			var bom cyclonedx.BOM
			err := json.Unmarshal(bomText, &bom)
			got, err := executeProcessing(tt.args.grypeLocation, bom)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeProcessing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("executeProcessing() got = %v, want %v", got, tt.want)
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
			numberOfProblems: 191
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
			if len(got) != tt.numberOfProblems) {
				t.Errorf("convertBOMToProblemsArray() got #problems %v, expected %v #problems", got, tt.want)
			}
		})
	}
}
