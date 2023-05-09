package handler

import (
	"encoding/json"
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

func Test_processFactsFromJSON(t *testing.T) {
	type args struct {
		facts         []byte
		environmentId int
		source        string
	}

	testResponse, err := ioutil.ReadFile("./testassets/factsPayload.json")
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

	tests := []struct {
		name string
		args args
		want []LagoonFact
	}{
		{
			name: "Process raw facts from JSON",
			args: args{
				b,
				90266,
				"drush-pml",
			},
			want: []LagoonFact{
				{
					Name:        "drupal-core",
					Value:       "9.0.1",
					Environment: 90266,
					Source:      "drush-pml",
					Description: "Drupal CMS version found on environment",
					KeyFact:     true,
					Type:        "TEXT",
				},
				{
					Name:        "php-version",
					Value:       "8.0.3",
					Environment: 90266,
					Source:      "drush-pml",
					Description: "PHP version found on environment",
					KeyFact:     false,
					Type:        "TEXT",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processFactsFromJSON(tt.args.facts, tt.args.source); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processFactsFromJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processFactsInsightsData(t *testing.T) {
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

	testResponse, err := ioutil.ReadFile("./testassets/testArrayFactsRawPayload.json")
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

	var tests = []struct {
		name    string
		args    args
		want    []LagoonFact
		want1   string
		wantErr bool
	}{
		{
			name: "raw facts payload",
			args: args{
				h: &Messaging{},
				insights: InsightsData{
					InputPayload: Payload,
					InsightsType: Raw,
					LagoonType:   Facts,
				},
				v:         string(data),
				apiClient: apiClient,
				resource: ResourceDestination{
					Project: "high-cotton",
					Service: "cli",
				},
			},
			want: []LagoonFact{
				{
					Environment: 3,
					Name:        "drupal-core",
					Value:       "9.0.1",
					Source:      "insights:facts:cli",
					Description: "Drupal CMS version found on environment",
					KeyFact:     true,
					Type:        "TEXT",
				},
			},
			want1:   "insights:facts:cli",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := processFactsInsightsData(tt.args.h, tt.args.insights, tt.args.v, tt.args.apiClient, tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("processFactsInsightsData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processFactsInsightsData() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("processFactsInsightsData() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
