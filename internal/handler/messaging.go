package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/cheshir/go-mq/v2"
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

	// Process s3 upload - that is, upload the incoming insights data to an s3 bucket
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

	// Check labels and annotations for insights data from message
	// use the environment initially
	if environment, ok := incoming.Labels["lagoon.sh/environment"]; ok {
		resource.Environment = environment
	}
	// override with branch annotation if provided
	if buildType, ok := incoming.Labels["lagoon.sh/buildType"]; ok && buildType != "pullrequest" {
		if branch, ok := incoming.Annotations["lagoon.sh/branch"]; ok {
			resource.Environment = branch
		}
	}
	if project, ok := incoming.Labels["lagoon.sh/project"]; ok {
		resource.Project = project
	}
	if service, ok := incoming.Labels["lagoon.sh/service"]; ok {
		resource.Service = service
	}
	if insightsType, ok := incoming.Labels["lagoon.sh/insightsType"]; ok {
		insights.InputType = insightsType
		if insightsType == "image-gz" {
			insights.LagoonType = ImageFacts
		}
	}
	if outputCompress, ok := incoming.Labels["lagoon.sh/insightsOutputCompressed"]; ok {
		compressed, _ := strconv.ParseBool(outputCompress)
		insights.OutputCompressed = compressed
	}
	if insightsOutputFileMIMEType, ok := incoming.Labels["lagoon.sh/insightsOutputFileMIMEType"]; ok {
		insights.OutputFileMIMEType = insightsOutputFileMIMEType
	}
	if insightsOutputFileExt, ok := incoming.Labels["lagoon.sh/insightsOutputFileExt"]; ok {
		insights.OutputFileExt = insightsOutputFileExt
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
