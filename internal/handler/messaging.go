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
}

// NewMessaging returns a messaging with config
func NewMessaging(config mq.Config, lagoonAPI LagoonAPI, s3 S3, startupAttempts int, startupInterval int, enableDebug bool, problemsFromSBOM bool, trivyServerEndpoint string) *Messaging {
	return &Messaging{
		Config:                  config,
		LagoonAPI:               lagoonAPI,
		S3Config:                s3,
		ConnectionAttempts:      startupAttempts,
		ConnectionRetryInterval: startupInterval,
		EnableDebug:             enableDebug,
		ProblemsFromSBOM:        problemsFromSBOM,
		TrivyServerEndpoint:     trivyServerEndpoint,
	}
}

// processMessageQueue reads in a rabbitMQ item and dispatches it to the appropriate function to process
func (h *Messaging) processMessageQueue(message mq.Message) {

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

	// here we unmarshal the initial incoming message body
	// notice how there is a "type" associated with the detail,
	// this is the primary driver used to determine which subsystem this message will be processed by.
	incoming := &InsightsMessage{}
	err := json.Unmarshal(message.Body(), incoming)

	if err != nil {
		fmt.Printf(err.Error())
		acknowledgeMessage()
		return
	}

	switch incoming.Type {
	case "direct.facts":
		resp := processFactsDirectly(message, h)
		slog.Debug(resp)
		acknowledgeMessage()
		return
	case "direct.problems":
		resp, _ := processProblemsDirectly(message, h)
		if h.EnableDebug {
			for _, d := range resp {
				slog.Debug(d)
			}
		}
		acknowledgeMessage()
		return
	case "direct.delete.problems":
		slog.Debug("Deleting problems")
		_, err := deleteProblemsDirectly(message, h)
		if err != nil {
			slog.Error(err.Error())
		}
		acknowledgeMessage() // Should we be acknowledging this error?
		return
	case "direct.delete.facts":
		_, err := deleteFactsDirectly(message, h)
		if err != nil {
			slog.Error(err.Error())
		}
		acknowledgeMessage() // Should we be acknowledging this error?
		return
	}

	// If we get here, we don't have an assigned type - which means we process the data via inferrence.
	// there are essentially two steps that happen there
	// First - we preprocess and clean up the incoming data
	// resource = contains details about where this came from
	// insights = contains details about the actual insights data itself
	resource, insights, err := preprocessIncomingMessageData(incoming)

	if err != nil {
		slog.Error("Error preprocessing - rejecting message and exiting", "Error", err.Error())
		rejectMessage(false)
	}

	slog.Debug("Insights", "data", fmt.Sprint(insights))
	slog.Debug("Target", "data", fmt.Sprint(resource))

	// Process s3 upload
	if !h.S3Config.Disabled {
		if insights.InsightsType != Direct {
			err := h.sendToLagoonS3(incoming, insights, resource)
			if err != nil {
				slog.Error("Unable to send to S3", "Error", err.Error())
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
			lagoonSourceFactMapCollection, err := h.gatherFactsFromInsightData(incoming, resource, insights)

			if err != nil {
				slog.Error("Unable to gather facts from incoming data", "Error", err.Error())
				rejectMessage(false)
				return
			}

			// Here we actually go ahead and write all the facts with their source
			for _, lsfm := range lagoonSourceFactMapCollection {
				for sourceName, facts := range lsfm {
					err := h.SendResultsetToLagoon(facts, resource, sourceName)
					if err != nil {
						slog.Error("Unable to write facts to api", "Error", err.Error())
						rejectMessage(false)
						return
					}
				}
			}

		}
	}
	acknowledgeMessage()
}

// preprocessIncomingMessageData deals with what are now legacy types, where most of the insight information
// used for further downstream processing is extracted from the message.
func preprocessIncomingMessageData(incoming *InsightsMessage) (ResourceDestination, InsightsData, error) {
	var resource ResourceDestination
	// Set some insight data defaults
	insights := InsightsData{
		LagoonType:         Facts,
		OutputFileExt:      "json",
		OutputFileMIMEType: "application/json",
	}

	// Check labels for insights data from message
	if incoming.Labels != nil {
		labelKeys := make([]string, 0, len(incoming.Labels))
		for k := range incoming.Labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)

		for _, label := range labelKeys {
			switch label {
			case "lagoon.sh/project":
				resource.Project = incoming.Labels["lagoon.sh/project"]
			case "lagoon.sh/environment":
				resource.Environment = incoming.Labels["lagoon.sh/environment"]
			case "lagoon.sh/service":
				resource.Service = incoming.Labels["lagoon.sh/service"]
			case "lagoon.sh/insightsType":
				insights.InputType = incoming.Labels["lagoon.sh/insightsType"]
				if incoming.Labels["lagoon.sh/insightsType"] == "image-gz" {
					insights.LagoonType = ImageFacts
				}
			case "lagoon.sh/insightsOutputCompressed":
				compressed, _ := strconv.ParseBool(incoming.Labels["lagoon.sh/insightsOutputCompressed"])
				insights.OutputCompressed = compressed
			case "lagoon.sh/insightsOutputFileMIMEType":
				insights.OutputFileMIMEType = incoming.Labels["lagoon.sh/insightsOutputFileMIMEType"]
			case "lagoon.sh/insightsOutputFileExt":
				insights.OutputFileExt = incoming.Labels["lagoon.sh/insightsOutputFileExt"]
			}
		}
	}

	// Define insights type from incoming 'insightsType' label
	if insights.InputType != "" {
		switch insights.InputType {
		case "sbom":
			return resource, insights, fmt.Errorf("insightsType of 'sbom' is deprecated, expect 'image-gz' - will not process")
		case "sbom-gz":
			insights.InsightsType = Sbom
		case "image":
			return resource, insights, fmt.Errorf("insightsType of 'image' is deprecated, expect 'image-gz' - will not process")
		case "image-gz":
			insights.InsightsType = Image
		case "direct":
			return resource, insights, fmt.Errorf("insightsType of 'direct' is deprecated, expect 'direct.facts' - will not process")
		default:
			insights.InsightsType = Raw
		}
	}

	// Determine incoming payload type
	if incoming.Payload == nil && incoming.BinaryPayload == nil {
		return resource, insights, fmt.Errorf("No payload was found")
	}
	if len(incoming.Payload) != 0 {
		insights.InputPayload = Payload
	}
	if len(incoming.BinaryPayload) != 0 {
		insights.InputPayload = BinaryPayload
	}

	return resource, insights, nil
}
