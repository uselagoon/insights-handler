package handler

import (
	"encoding/json"
	"fmt"
	"github.com/cheshir/go-mq"
	"log"
	"sort"
	"strconv"
)

// Messaging is used for the config and client information for the messaging queue, including processing the queue itself.
type Messaging struct {
	Config                  mq.Config
	LagoonAPI               LagoonAPI
	S3Config                S3
	ConnectionAttempts      int
	ConnectionRetryInterval int
	EnableDebug             bool
	ProblemsFromSBOM        bool
	GrypeBinaryLocation     string
}

// NewMessaging returns a messaging with config
func NewMessaging(config mq.Config, lagoonAPI LagoonAPI, s3 S3, startupAttempts int, startupInterval int, enableDebug bool, problemsFromSBOM bool, grypeBinaryLocation string) *Messaging {
	return &Messaging{
		Config:                  config,
		LagoonAPI:               lagoonAPI,
		S3Config:                s3,
		ConnectionAttempts:      startupAttempts,
		ConnectionRetryInterval: startupInterval,
		EnableDebug:             enableDebug,
		ProblemsFromSBOM:        problemsFromSBOM,
		GrypeBinaryLocation:     grypeBinaryLocation,
	}
}

// processMessageQueue reads in a rabbitMQ item and dispatches it to the appropriate function to process
func (h *Messaging) processMessageQueue(message mq.Message) {
	var insights InsightsData
	var resource ResourceDestination

	// set up defer to ack the message after we're done processing
	defer func(message mq.Message) {
		// Ack to remove from queue
		err := message.Ack(false)
		if err != nil {
			fmt.Printf("Failed to acknowledge message: %s\n", err.Error())
		}
	}(message)

	incoming := &InsightsMessage{}
	json.Unmarshal(message.Body(), incoming)

	// if we have direct problems or facts, we process them differently - skipping all
	// the extra processing below.
	if incoming.Type == "direct.facts" || incoming.Type == "direct.problems" {
		resp := processItemsDirectly(message, h)
		log.Println(resp)
		return
	}

	// Check labels for insights data from message
	if incoming.Labels != nil {
		labelKeys := make([]string, 0, len(incoming.Labels))
		for k := range incoming.Labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)

		// Set some insight data defaults
		insights = InsightsData{
			LagoonType:         Facts,
			OutputFileExt:      "json",
			OutputFileMIMEType: "application/json",
		}

		for _, label := range labelKeys {
			if label == "lagoon.sh/project" {
				resource.Project = incoming.Labels["lagoon.sh/project"]
			}
			if label == "lagoon.sh/environment" {
				resource.Environment = incoming.Labels["lagoon.sh/environment"]
			}
			if label == "lagoon.sh/service" {
				resource.Service = incoming.Labels["lagoon.sh/service"]
			}

			if label == "lagoon.sh/insightsType" {
				insights.InputType = incoming.Labels["lagoon.sh/insightsType"]
			}
			if incoming.Labels["lagoon.sh/insightsType"] == "image-gz" {
				insights.LagoonType = ImageFacts
			}
			if label == "lagoon.sh/insightsOutputCompressed" {
				compressed, _ := strconv.ParseBool(incoming.Labels["lagoon.sh/insightsOutputCompressed"])
				insights.OutputCompressed = compressed
			}
			if label == "lagoon.sh/insightsOutputFileMIMEType" {
				insights.OutputFileMIMEType = incoming.Labels["lagoon.sh/insightsOutputFileMIMEType"]
			}
			if label == "lagoon.sh/insightsOutputFileExt" {
				insights.OutputFileExt = incoming.Labels["lagoon.sh/insightsOutputFileExt"]
			}
		}
	}

	// Define insights type from incoming 'insightsType' label
	if insights.InputType != "" {
		switch insights.InputType {
		case "sbom", "sbom-gz":
			insights.InsightsType = Sbom
		case "image", "image-gz":
			insights.InsightsType = Image
		case "direct":
			insights.InsightsType = Direct
		default:
			insights.InsightsType = Raw
		}
	}

	// Determine incoming payload type
	if incoming.Payload == nil && incoming.BinaryPayload == nil {
		if h.EnableDebug {
			log.Printf("[DEBUG] no payload was found")
		}
		err := message.Reject(false)
		if err != nil {
			fmt.Printf("Unable to reject payload: %s\n", err.Error())
		}
		return
	}
	if len(incoming.Payload) != 0 {
		insights.InputPayload = Payload
	}
	if len(incoming.BinaryPayload) != 0 {
		insights.InputPayload = BinaryPayload
	}

	// Debug
	if h.EnableDebug {
		log.Println("[DEBUG] insights:", insights)
		log.Println("[DEBUG] target:", resource)
	}

	// Process s3 upload
	if !h.S3Config.Disabled {
		if insights.InsightsType != Direct {
			err := h.sendToLagoonS3(incoming, insights, resource)
			if err != nil {
				log.Printf("Unable to send to S3: %s", err.Error())
			}
		}
	}

	// Process Lagoon API integration
	if !h.LagoonAPI.Disabled {
		if insights.InsightsType != Sbom &&
			insights.InsightsType != Image &&
			insights.InsightsType != Raw &&
			insights.InsightsType != Direct {
			log.Println("only 'sbom', 'direct', 'raw', and 'image' types are currently supported for api processing")
		} else {
			err := h.sendToLagoonAPI(incoming, resource, insights)
			if err != nil {
				log.Printf("Unable to send to the api: %s", err.Error())
			}
		}
	}
}
