package handler

import (
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
	"github.com/cheshir/go-mq"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/mock"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
)

type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) Ack(multiple bool) error {
	args := m.Called(multiple)
	return args.Error(0)
}

func (m *MockMessage) Nack(multiple, request bool) error {
	args := m.Called(multiple, request)
	return args.Error(0)
}

func (m *MockMessage) Reject(requeue bool) error {
	args := m.Called(requeue)
	return args.Error(0)
}

func (m *MockMessage) Body() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *MockMessage) AppId() string {
	args := m.Called()
	return args.String(0)
}

func Test_processingIncomingMessageQueue(t *testing.T) {
	type args struct {
		message mq.Message
	}
	var tests []struct {
		name string
		args args
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_processDirectFacts(t *testing.T) {
	err := godotenv.Load("../../.env.example")
	if err != nil {
		fmt.Println(err)
		panic("Error loading .env file")

	}

	tokenSigningKey := os.Getenv("TOKEN_SIGNING_KEY")
	if tokenSigningKey == "" {
		fmt.Println("TokenSigningKey not found in environment variables")
		return
	}

	h := Messaging{
		Config: mq.Config{},
		LagoonAPI: LagoonAPI{
			Endpoint:        "http://localhost:3000/graphql",
			TokenSigningKey: tokenSigningKey,
			JWTAudience:     "api.dev",
		},
	}
	// apiClient := h.getApiClient()

	testResponse, err := ioutil.ReadFile("./testassets/directFactsPayload.json")
	if err != nil {
		t.Fatalf("Could not open file: %v", err)
	}

	type args struct {
		message mq.Message
		h       *Messaging
	}

	var tests = []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "direct facts insights payload",
			args: args{
				message: &MockMessage{},
				h:       &h,
			},
			want:    "Added 2 fact(s)",
			want1:   "insights:facts:cli",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.args.message.(*MockMessage)

			// Set up the expected behavior of the mock message's Body and Ack methods
			message.On("Body").Return(testResponse)
			message.On("Ack", false).Return(nil)

			fmt.Println(string(message.Body()))

			got := processFactsDirectly(tt.args.message, tt.args.h)
			if (err != nil) != tt.wantErr {
				t.Errorf("processFactsDirectly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processFactsDirectly() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processFactsFromSBOM(t *testing.T) {
	type args struct {
		bom           *[]cdx.Component
		environmentId int
		source        string
	}

	testResponse, err := ioutil.ReadFile("./testassets/testSbomPayload.json")
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

	bom := new(cdx.BOM)
	resp, err := http.Get(server.URL)
	if err != nil {
		panic(err)
	}
	decoder := cdx.NewBOMDecoder(resp.Body, cdx.BOMFileFormatJSON)
	if err = decoder.Decode(bom); err != nil {
		panic(err)
	}

	tests := []struct {
		name string
		args args
		want []lagoonclient.AddFactInput
	}{
		{
			name: "sbom.cdx.json",
			args: args{
				bom:           bom.Components,
				environmentId: 3,
				source:        "syft",
			},
			want: []lagoonclient.AddFactInput{
				{
					Environment: 3,
					Name:        "@npmcli/arborist",
					Value:       "2.6.2",
					Source:      "syft",
					Description: "pkg:npm/@npmcli%2Farborist@2.6.2",
					KeyFact:     false,
					Type:        "TEXT",
				},
				{
					Environment: 3,
					Name:        "@npmcli/ci-detect",
					Value:       "1.3.0",
					Source:      "syft",
					Description: "pkg:npm/@npmcli%2Fci-detect@1.3.0",
					KeyFact:     false,
					Type:        "TEXT",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processFactsFromSBOM(slog.Default(), tt.args.bom, tt.args.environmentId, tt.args.source)
			if len(got) != len(tt.want) {
				t.Errorf("processFactsFromSBOM() returned %d results, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i].Environment != tt.want[i].Environment ||
					got[i].Name != tt.want[i].Name ||
					got[i].Value != tt.want[i].Value ||
					got[i].Source != tt.want[i].Source ||
					got[i].Description != tt.want[i].Description ||
					got[i].KeyFact != tt.want[i].KeyFact {
					t.Errorf("processFactsFromSBOM()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
