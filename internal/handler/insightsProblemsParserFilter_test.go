package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/cheshir/go-mq"
)

func Test_processProblemsInsightsData(t *testing.T) {
	type args struct {
		h         *Messaging
		insights  InsightsData
		v         string
		apiClient graphql.Client
		resource  ResourceDestination
	}

	h := Messaging{
		Config: mq.Config{},
		LagoonAPI: LagoonAPI{
			Endpoint: "http://localhost:3000/graphql",
		},
	}
	apiClient := h.getApiClient()

	testResponse, err := ioutil.ReadFile("./testassets/trivyVulnReportPayload.json")
	if err != nil {
		t.Fatalf("Could not open file")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Expected to request '/fixedvalue', got: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(testResponse)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	incoming := &InsightsMessage{}
	json.Unmarshal(b, incoming)

	var payload PayloadInput
	for _, p := range incoming.Payload {
		payload = p
	}

	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name    string
		args    args
		want    []LagoonProblem
		want1   string
		wantErr bool
	}{
		{
			name: "trivy vuln report payload",
			args: args{
				h: &Messaging{},
				insights: InsightsData{
					InputType:    "trivy-vuln-report",
					InputPayload: Payload,
					InsightsType: Raw,
					LagoonType:   Problems,
				},
				v:         string(data),
				apiClient: apiClient,
				resource: ResourceDestination{
					Project: "high-cotton",
					Service: "nginx",
				},
			},
			want: []LagoonProblem{
				{
					Environment:       3,
					Identifier:        "CVE-2020-8911",
					AssociatedPackage: "github.com/aws/aws-sdk-go",
					Version:           "v1.34.21",
					FixedVersion:      "v1.34.22",
					Source:            "insights:problems:trivy-vuln-report",
					Service:           "nginx",
					Data:              "{}",
					Severity:          "MEDIUM",
					Description:       "aws/aws-sdk-go: CBC padding oracle issue in AWS S3 Crypto SDK for golang",
					Links:             "https://avd.aquasec.com/nvd/cve-2020-8911",
					SeverityScore:     0.56,
				},
				{
					Environment:       3,
					Identifier:        "CVE-2020-8912",
					AssociatedPackage: "github.com/aws/aws-sdk-go",
					Version:           "v1.34.21",
					FixedVersion:      "v1.34.22",
					Source:            "insights:problems:trivy-vuln-report",
					Service:           "nginx",
					Data:              "{}",
					Severity:          "LOW",
					Description:       "aws-sdk-go: In-band key negotiation issue in AWS S3 Crypto SDK for golang",
					Links:             "https://avd.aquasec.com/nvd/cve-2020-8912",
					SeverityScore:     0.25,
				},
				{
					Environment:       3,
					Identifier:        "CVE-2022-28923",
					AssociatedPackage: "github.com/caddyserver/caddy",
					Version:           "v1.0.5",
					FixedVersion:      "v2.5.0",
					Source:            "insights:problems:trivy-vuln-report",
					Service:           "nginx",
					Data:              "{}",
					Severity:          "MEDIUM",
					Description:       "caddy: an open redirection vulnerability which allows attackers to redirect users to phishing websites via crafted URLs",
					Links:             "https://avd.aquasec.com/nvd/cve-2022-28923",
					SeverityScore:     0.75,
				},
			},
			want1:   "insights:problems:trivy-vuln-report",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := processProblemsInsightsData(tt.args.h, tt.args.insights, tt.args.v, tt.args.apiClient, tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("processProblemsInsightsData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				fmt.Printf("got: %#v\n", got)
				fmt.Printf("want: %#v\n", tt.want)
				t.Errorf("processProblemsInsightsData() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("processProblemsInsightsData() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_processDirectProblemsInsightsData(t *testing.T) {
	type args struct {
		h         *Messaging
		insights  InsightsData
		v         string
		apiClient graphql.Client
		resource  ResourceDestination
	}

	h := Messaging{
		Config: mq.Config{},
		LagoonAPI: LagoonAPI{
			Endpoint: "http://localhost:3000/graphql",
		},
	}
	apiClient := h.getApiClient()

	testResponse, err := ioutil.ReadFile("./testassets/testDirectProblems.json")
	if err != nil {
		t.Fatalf("Could not open file")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("Expected to request '/fixedvalue', got: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(testResponse)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	incoming := &InsightsMessage{}
	json.Unmarshal(b, incoming)

	var payload PayloadInput
	for _, p := range incoming.Payload {
		payload = p
	}

	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name    string
		args    args
		want    []LagoonProblem
		want1   string
		wantErr bool
	}{
		{
			name: "direct problems payload",
			args: args{
				h: &Messaging{},
				insights: InsightsData{
					InputType:    "direct",
					InputPayload: Payload,
					InsightsType: Direct,
					LagoonType:   Problems,
				},
				v:         string(data),
				apiClient: apiClient,
				resource: ResourceDestination{
					Project: "high-cotton",
					Service: "nginx",
				},
			},
			want: []LagoonProblem{
				{
					Environment:       3,
					Identifier:        "CVE-2020-8911",
					AssociatedPackage: "github.com/aws/aws-sdk-go",
					Version:           "v1.34.21",
					FixedVersion:      "v1.34.22",
					Source:            "insights:problems:direct",
					Service:           "go",
					Data:              "{}",
					Severity:          "MEDIUM",
					Description:       "aws/aws-sdk-go: CBC padding oracle issue in AWS S3 Crypto SDK for golang",
					Links:             "https://avd.aquasec.com/nvd/cve-2020-8911",
					SeverityScore:     0.56,
				},
				{
					Environment:       3,
					Identifier:        "CVE-2020-8912",
					AssociatedPackage: "github.com/aws/aws-sdk-go",
					Version:           "v1.44.21",
					FixedVersion:      "v1.44.22",
					Source:            "insights:problems:direct",
					Service:           "go",
					Data:              "{}",
					Severity:          "LOW",
					Description:       "aws-sdk-go: In-band key negotiation issue in AWS S3 Crypto SDK for golang",
					Links:             "https://avd.aquasec.com/nvd/cve-2020-8912",
					SeverityScore:     0.25,
				},
			},
			want1:   "insights:problems:direct",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := processProblemsInsightsData(tt.args.h, tt.args.insights, tt.args.v, tt.args.apiClient, tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("processProblemsInsightsData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				fmt.Printf("got: %#v\n", got)
				fmt.Printf("want: %#v\n", tt.want)
				t.Errorf("processProblemsInsightsData() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("processProblemsInsightsData() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
