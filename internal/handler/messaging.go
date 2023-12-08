package handler

import (
	"encoding/json"
	"fmt"
	"github.com/cheshir/go-mq"
	"log/slog"
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
	TrivyServerEndpoint     string
	MessageQWriter          func(data []byte) error
}

// NewMessaging returns a messaging with config
func NewMessaging(config mq.Config, lagoonAPI LagoonAPI, s3 S3, startupAttempts int, startupInterval int, enableDebug bool, problemsFromSBOM bool, trivyServerEndpoint string, MessageQWriter func(data []byte) error) *Messaging {
	return &Messaging{
		Config:                  config,
		LagoonAPI:               lagoonAPI,
		S3Config:                s3,
		ConnectionAttempts:      startupAttempts,
		ConnectionRetryInterval: startupInterval,
		EnableDebug:             enableDebug,
		ProblemsFromSBOM:        problemsFromSBOM,
		TrivyServerEndpoint:     trivyServerEndpoint,
		MessageQWriter:          MessageQWriter,
	}
}

// processMessageQueue reads in a rabbitMQ item and dispatches it to the appropriate function to process
func (h *Messaging) processMessageQueue(message mq.Message) {
	var insights InsightsData
	var resource ResourceDestination
	acknowledgeMessage := func(message mq.Message) func() {
		return func() {
			// Ack to remove from queue
			err := message.Ack(false)
			if err != nil {
				slog.Error("Failed to acknowledge message", "Error", err.Error())
			}
		}
	}(message)

	rejectMessage := func(message mq.Message) func(bool) {
		return func(requeue bool) {
			// Ack to remove from queue
			err := message.Reject(requeue)
			if err != nil {
				slog.Error("Failed to reject message", "Error", err.Error())
			}
		}
	}(message)

	incoming := &InsightsMessage{}
	err := json.Unmarshal(message.Body(), incoming)

	if err != nil {
		fmt.Printf(err.Error())
		acknowledgeMessage()
		return
	}

	// if we have direct problems or facts, we process them differently - skipping all
	// the extra processing below.
	if incoming.Type == "direct.facts" {
		resp := processFactsDirectly(message, h)
		slog.Debug(resp)
		acknowledgeMessage()
		return
	}

	if incoming.Type == "direct.problems" {
		resp, _ := processProblemsDirectly(message, h)
		if h.EnableDebug {
			for _, d := range resp {
				slog.Debug(d)
			}
		}
		acknowledgeMessage()
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
		slog.Debug("No payload was found - rejecting message and exiting")
		rejectMessage(false)
		return
	}
	if len(incoming.Payload) != 0 {
		insights.InputPayload = Payload
	}
	if len(incoming.BinaryPayload) != 0 {
		insights.InputPayload = BinaryPayload
	}

	// Debug
	//if h.EnableDebug {
	//	log.Println("[DEBUG] insights:", insights)
	//	log.Println("[DEBUG] target:", resource)
	//}
	slog.Debug("Insights", "data", fmt.Sprint(insights))
	slog.Debug("Target", "data", fmt.Sprint(resource))

	// Process s3 upload
	if !h.S3Config.Disabled {
		if insights.InsightsType != Direct {
			err := h.sendToLagoonS3(incoming, insights, resource)
			if err != nil {
				incoming.RequeueAttempts++
				updatedMessage, err := json.Marshal(incoming)
				if err != nil {
					fmt.Printf(err.Error())
				}
				if incoming.RequeueAttempts <= 3 {
					rejectMessage(false)
					if err := h.MessageQWriter(updatedMessage); err != nil {
						slog.Error("Error re-queueing message", "Error", err.Error())
					}
					return
				} else {
					slog.Error("Retries failed, unable to send to S3", "Error", err.Error())
					rejectMessage(false)
					return
				}
			}
		}
	}

	// Process Lagoon API integration
	if !h.LagoonAPI.Disabled {
		if insights.InsightsType != Sbom &&
			insights.InsightsType != Image &&
			insights.InsightsType != Raw &&
			insights.InsightsType != Direct {
			slog.Error("only 'sbom', 'direct', 'raw', and 'image' types are currently supported for api processing")
		} else {
			err := h.sendToLagoonAPI(incoming, resource, insights)

			if err != nil {
				incoming.RequeueAttempts++
				updatedMessage, err := json.Marshal(incoming)
				if err != nil {
					fmt.Printf(err.Error())
				}
				if incoming.RequeueAttempts <= 3 {
					rejectMessage(false)
					if err := h.MessageQWriter(updatedMessage); err != nil {
						slog.Error("Error re-queueing message", "Error", err.Error())
					}
					return
				} else {
					slog.Error("Retries failed, unable to send to the API", "Error", err.Error())
					rejectMessage(false)
					return
				}
			}
		}
	}
	acknowledgeMessage()
}
