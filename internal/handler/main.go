package handler

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/cheshir/go-mq"
	"github.com/matryer/try"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/lagoonclient/jwt"
)

var EnableDebug bool

// RabbitBroker .
type RabbitBroker struct {
	Hostname     string `json:"hostname"`
	Port         string `json:"port"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	QueueName    string `json:"queueName"`
	ExchangeName string `json:"exchangeName"`
}

// LagoonAPI .
type LagoonAPI struct {
	Endpoint        string `json:"endpoint"`
	JWTAudience     string `json:"audience"`
	TokenSigningKey string `json:"tokenSigningKey"`
	JWTSubject      string `json:"subject"`
	JWTIssuer       string `json:"issuer"`
	Disabled        bool   `json:"disableApiIntegration"`
}

// S3 Config .
type S3 struct {
	SecretAccessKey string `json:"secretAccessKey"`
	S3Origin        string `json:"s3Origin"`
	AccessKeyId     string `json:"accessKeyId"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	UseSSL          bool   `json:"useSSL"`
	Disabled        bool   `json:"disableS3upload"`
}

type InsightsMessage struct {
	Payload       []PayloadInput    `json:"payload"`
	BinaryPayload map[string]string `json:"binaryPayload"`
	Annotations   map[string]string `json:"annotations"`
	Labels        map[string]string `json:"labels"`
	Type          string            `json:"type,omitempty"`
}

type PayloadInput struct {
	Project     string       `json:"project,omitempty"`
	Environment string       `json:"environment,omitempty"`
	Facts       []LagoonFact `json:"facts,omitempty"`
}

type DirectFact struct {
	EnvironmentId   json.Number `json:"environment"`
	ProjectName     string      `json:"projectName"`
	EnvironmentName string      `json:"environmentName"`
	Name            string      `json:"name"`
	Value           string      `json:"value"`
	Description     string      `json:"description"`
	Type            string      `json:"type"`
	Category        string      `json:"category"`
	Service         string      `json:"service"`
	Source          string      `json:"source"`
}

type DirectFacts struct {
	ProjectName     string       `json:"projectName,omitempty"`
	EnvironmentName string       `json:"environmentName,omitempty"`
	EnvironmentId   json.Number  `json:"environment,omitempty"`
	Facts           []DirectFact `json:"facts"`
	Type            string       `json:"type"`
	InsightsType    string       `json:"insightsType"`
}

type DirectProblems struct {
	EnvironmentId   int                          `json:"environment"`
	ProjectName     string                       `json:"projectName"`
	EnvironmentName string                       `json:"environmentName"`
	Problems        []lagoonclient.LagoonProblem `json:"problems"`
	Type            string                       `json:"type"`
}

type DirectDeleteMessage struct {
	Type          string `json:"type"`
	EnvironmentId int    `json:"environment"`
	Source        string `json:"source"`
	Service       string `json:"service"`
}

type InsightsData struct {
	InputType               string
	InputPayload            PayloadType
	InsightsType            InsightType
	InsightsCompressionType string
	InsightsFileType        string
	LagoonType              LagoonType
	OutputFileExt           string
	OutputFileMIMEType      string
	OutputCompressed        bool
}

// LagoonFact Here we wrap outgoing facts before passing them to genqlient
type LagoonFact struct {
	Id          int    `json:"id"`
	Environment int    `json:"environment"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Source      string `json:"source"`
	Description string `json:"description"`
	KeyFact     bool   `json:"keyFact"`
	Type        string `json:"type"`
	Category    string `json:"category"`
}

const (
	FactTypeText   string = "TEXT"
	FactTypeUrl    string = "URL"
	FactTypeSemver string = "SEMVER"
)

type InsightType int64

const (
	Raw = iota
	Sbom
	Image
	Direct
)

func (i InsightType) String() string {
	switch i {
	case Raw:
		return "RAW"
	case Sbom:
		return "SBOM"
	case Image:
		return "IMAGE"
	case Direct:
		return "DIRECT"
	}
	return "RAW"
}

type LagoonType int64

const (
	Facts = iota
	ImageFacts
	Problems
)

func (t LagoonType) String() string {
	switch t {
	case Facts:
		return "FACTS"
	case ImageFacts:
		return "IMAGE"
	case Problems:
		return "PROBLEMS"
	}
	return "UNKNOWN"
}

type PayloadType int64

const (
	Payload = iota
	BinaryPayload
)

func (p PayloadType) String() string {
	switch p {
	case Payload:
		return "PAYLOAD"
	case BinaryPayload:
		return "BINARY_PAYLOAD"
	}
	return "PAYLOAD"
}

type EnvironmentVariable struct {
	Key   string
	Value string
}

type ResourceDestination struct {
	Project     string
	Environment string
	Service     string
	Format      string
}

// Consumer handles consuming messages sent to the queue that this action handler is connected to and processes them accordingly
func (h *Messaging) Consumer() {
	var messageQueue mq.MQ

	// if no mq is found when the goroutine starts, retry a few times before exiting
	// default is 10 retry with 30 second delay = 5 minutes
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		messageQueue, err = mq.New(h.Config)
		if err != nil {
			slog.Error(fmt.Sprintf(
				"Failed to initialize message queue manager, retrying in %d seconds, attempt %d/%d",
				h.ConnectionRetryInterval,
				attempt,
				h.ConnectionAttempts,
			),
				"error", err.Error(),
			)
			time.Sleep(time.Duration(h.ConnectionRetryInterval) * time.Second)
		}
		return attempt < h.ConnectionAttempts, err
	})
	if err != nil {
		//log.Fatalf("Finally failed to initialize message queue manager: %v", err)
		slog.Error("Finally failed to initialize message queue manager", "error", err.Error())
		os.Exit(1)
	}
	defer messageQueue.Close()

	go func() {
		for err := range messageQueue.Error() {
			//log.Println(fmt.Sprintf("Caught error from message queue: %v", err))
			slog.Error("Caught error from message queue", "Error", err.Error())
		}
	}()

	forever := make(chan bool)

	// Handle any tasks that go to the queue
	//log.Println("Listening for messages in queue lagoon-insights:items")
	slog.Info("Listening for messages", "queue", "lagoon-insights:items")
	err = messageQueue.SetConsumerHandler("items-queue", h.processMessageQueue)
	if err != nil {
		//log.Println(fmt.Sprintf("Failed to set handler to consumer `%s`: %v", "items-queue", err))
		slog.Error("Failed to set handler", "consumer", "items-queue", "error", err.Error())
	}
	<-forever
}

type authedTransport struct {
	wrapped http.RoundTripper
	h       *Messaging
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	//grab events for project
	token, err := jwt.OneMinuteAdminToken(t.h.LagoonAPI.TokenSigningKey, t.h.LagoonAPI.JWTAudience, t.h.LagoonAPI.JWTSubject, t.h.LagoonAPI.JWTIssuer)
	if err != nil {
		// the token wasn't generated
		slog.Debug("Error while creating JWT", "error", err.Error())
		return nil, err
	}
	req.Header.Set("Authorization", "bearer "+token)
	return t.wrapped.RoundTrip(req)
}

type LagoonSourceFactMap map[string][]LagoonFact

// Incoming payload may contain facts or problems, so we need to handle these differently
func (h *Messaging) gatherFactsFromInsightData(incoming *InsightsMessage, resource ResourceDestination, insights InsightsData) ([]LagoonSourceFactMap, error) {
	apiClient := h.getApiClient()
	// Here we collect all source fact maps before writing them _once_
	lagoonSourceFactMapCollection := []LagoonSourceFactMap{}

	if resource.Project == "" && resource.Environment == "" {
		return lagoonSourceFactMapCollection, fmt.Errorf("no resource definition labels could be found in payload (i.e. lagoon.sh/project or lagoon.sh/environment)")
	}

	if insights.InputPayload == Payload && insights.LagoonType == Facts {
		for _, p := range incoming.Payload {
			lagoonSourceFactMap, err := parserFilterLoopForPayloads(insights, p, h, apiClient, resource)
			if err != nil {
				return lagoonSourceFactMapCollection, err
			}
			lagoonSourceFactMapCollection = append(lagoonSourceFactMapCollection, lagoonSourceFactMap)
		}
	}

	if insights.InputPayload == BinaryPayload && insights.LagoonType == Facts {
		for _, p := range incoming.BinaryPayload {
			lagoonSourceFactMap, err := parserFilterLoopForBinaryPayloads(insights, p, h, apiClient, resource)
			if err != nil {
				return lagoonSourceFactMapCollection, err
			}
			lagoonSourceFactMapCollection = append(lagoonSourceFactMapCollection, lagoonSourceFactMap)
		}
	}

	return lagoonSourceFactMapCollection, nil
}

func parserFilterLoopForBinaryPayloads(insights InsightsData, p string, h *Messaging, apiClient graphql.Client, resource ResourceDestination) (LagoonSourceFactMap, error) {
	lagoonSourceFactMap := LagoonSourceFactMap{}
	for _, filter := range parserFilters {

		result, source, err := filter(h, insights, p, apiClient, resource)
		if err != nil {
			slog.Error("Error running filter", "error", err.Error())
			return lagoonSourceFactMap, err
		}
		lagoonSourceFactMap[source] = result
	}
	return lagoonSourceFactMap, nil
}

func parserFilterLoopForPayloads(insights InsightsData, p PayloadInput, h *Messaging, apiClient graphql.Client, resource ResourceDestination) (LagoonSourceFactMap, error) {
	lagoonSourceFactMap := LagoonSourceFactMap{}
	for _, filter := range parserFilters {
		var result []LagoonFact
		var source string

		json, err := json.Marshal(p)
		if err != nil {
			slog.Error("Error marshalling data", "error", err.Error())
			return lagoonSourceFactMap, err
		}

		result, source, err = filter(h, insights, fmt.Sprintf("%s", json), apiClient, resource)
		if err != nil {
			slog.Error("Error Filtering payload", "error", err.Error())
			return lagoonSourceFactMap, err
		}
		lagoonSourceFactMap[source] = result
	}
	return lagoonSourceFactMap, nil
}

func trivySBOMProcessing(apiClient graphql.Client, trivyServerEndpoint string, resource ResourceDestination, payload string) error {

	bom, err := getBOMfromPayload(payload)
	if err != nil {
		return err
	}

	// Determine lagoon resource destination
	_, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		return apiErr
	}

	// we process the SBOM here
	// TODO: This should actually live in its own function somewhere else.
	isAlive, err := IsTrivyServerIsAlive(trivyServerEndpoint)
	if err != nil {
		return fmt.Errorf("trivy server not alive: %v", err.Error())
	} else {
		slog.Debug("Trivy is reachable")
	}
	if isAlive {
		err = SbomToProblems(apiClient, trivyServerEndpoint, "/tmp/", environment.Id, resource.Service, *bom)
	}
	if err != nil {
		return err
	}
	return nil
}

// sendResultsetToLagoon will send results as facts to the lagoon api after processing via a parser filter
func (h *Messaging) SendResultsetToLagoon(result []LagoonFact, resource ResourceDestination, source string) error {
	apiClient := h.getApiClient()
	project, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		slog.Error(apiErr.Error())
		return apiErr
	}

	// Even if we don't find any new facts, we need to delete the existing ones
	// since these may be the end product of a filter process
	apiErr = h.deleteExistingFactsBySource(apiClient, environment, source, project)
	if apiErr != nil {
		slog.Error(apiErr.Error())
		return apiErr
	}

	e := h.sendFactsToLagoonAPI(result, apiClient, resource, source)
	if e != nil {
		slog.Error("Error sending facts to Lagoon API", "error", e.Error())
		return e
	}

	return nil
}

func (h *Messaging) sendFactsToLagoonAPI(facts []LagoonFact, apiClient graphql.Client, resource ResourceDestination, source string) error {

	slog.Debug("Matched facts",
		"Number", len(facts),
		"ProjectName", resource.Project,
		"EnvironmentId", resource.Environment,
		"Source", source,
	)

	if len(facts) > 0 {
		apiErr := h.pushFactsToLagoonApi(facts, resource)
		if apiErr != nil {
			return fmt.Errorf("%s", apiErr.Error())
		}
	}

	return nil
}

func (h *Messaging) deleteExistingFactsBySource(apiClient graphql.Client, environment lagoonclient.Environment, source string, project lagoonclient.Project) error {
	// Remove existing facts from source first
	_, err := lagoonclient.DeleteFactsFromSource(context.TODO(), apiClient, environment.Id, source)
	if err != nil {
		return err
	}

	slog.Info("Previous facts deleted",
		"ProjectId", project.Id,
		"ProjectName", project.Name,
		"EnvironmentId", environment.Id,
		"EnvironmentName", environment.Name,
		"Source", source,
	)

	return nil
}

func (h *Messaging) getApiClient() graphql.Client {
	apiClient := graphql.NewClient(h.LagoonAPI.Endpoint, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, h: h}})
	return apiClient
}

func determineResourceFromLagoonAPI(apiClient graphql.Client, resource ResourceDestination) (lagoonclient.Project, lagoonclient.Environment, error) {
	// Get project data (we need the project ID to be able to utilise the environmentByName query)
	project, err := lagoonclient.GetProjectByName(context.TODO(), apiClient, resource.Project)
	if err != nil {
		return lagoonclient.Project{}, lagoonclient.Environment{}, fmt.Errorf("error: unable to determine resource destination (does %s:%s exist?): %v", resource.Project, resource.Environment, err.Error())
	}

	if project.Id == 0 || project.Name == "" {
		return lagoonclient.Project{}, lagoonclient.Environment{}, fmt.Errorf("error: unable to determine resource destination (does %s:%s exist?): %v", resource.Project, resource.Environment, err.Error())
	}

	environment, err := lagoonclient.GetEnvironmentFromName(context.TODO(), apiClient, resource.Environment, project.Id)
	if err != nil {
		return lagoonclient.Project{}, lagoonclient.Environment{}, err
	}
	return project, environment, nil
}

func (h *Messaging) sendToLagoonS3(incoming *InsightsMessage, insights InsightsData, resource ResourceDestination) (err error) {
	// strip http/s protocol from origin
	u, _ := url.Parse(h.S3Config.S3Origin)
	if u.Scheme == "http" || u.Scheme == "https" {
		h.S3Config.S3Origin = u.Host
	}

	// Push to s3 bucket
	minioClient, err := minio.New(h.S3Config.S3Origin, &minio.Options{
		Creds:  credentials.NewStaticV4(h.S3Config.AccessKeyId, h.S3Config.SecretAccessKey, ""),
		Secure: h.S3Config.UseSSL,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = minioClient.MakeBucket(ctx, h.S3Config.Bucket, minio.MakeBucketOptions{Region: h.S3Config.Region})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, h.S3Config.Bucket)
		if errBucketExists != nil && !exists {
			return err
		}
	} else {
		slog.Info(fmt.Sprintf("Successfully created %s", h.S3Config.Bucket))
	}

	if len(incoming.Payload) != 0 {
		b, err := json.Marshal(incoming)
		if err != nil {
			return err
		}

		objectName := strings.ToLower(fmt.Sprintf("%s-%s-%s-%s.json", insights.InsightsType, resource.Project, resource.Environment, resource.Service))
		reader := bytes.NewReader(b)

		info, putObjErr := minioClient.PutObject(ctx, h.S3Config.Bucket, objectName, reader, reader.Size(), minio.PutObjectOptions{
			ContentEncoding: "application/json",
		})
		if putObjErr != nil {
			return putObjErr
		}

		slog.Info(fmt.Sprintf("Successfully uploaded %s of size %d", objectName, info.Size))
	}

	if len(incoming.BinaryPayload) != 0 {
		for _, p := range incoming.BinaryPayload {
			result, err := decodeGzipString(p)
			if err != nil {
				return err
			}
			resultJson, _ := json.MarshalIndent(result, "", " ")

			fileExt := insights.OutputFileExt
			contentType := insights.OutputFileMIMEType
			var contentEncoding string
			if insights.OutputCompressed == true {
				fileExt = fmt.Sprintf("%s.gz", insights.OutputFileExt)
				contentEncoding = "gzip"
			}

			objectName := strings.ToLower(fmt.Sprintf("%s-%s-%s-%s.%s", insights.InsightsType, resource.Project, resource.Environment, resource.Service, fileExt))
			tempFilePath := fmt.Sprintf("/tmp/%s", objectName)

			if insights.OutputCompressed != true {
				err = ioutil.WriteFile(tempFilePath, resultJson, 0644)
				if err != nil {
					return err
				}
			} else {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				gz.Write(resultJson)
				gz.Close()
				err = ioutil.WriteFile(tempFilePath, buf.Bytes(), 0644)
				if err != nil {
					return err
				}
			}

			s3FilePath := strings.ToLower(fmt.Sprintf("insights/%s/%s/%s", resource.Project, resource.Environment, objectName))
			info, err := minioClient.FPutObject(ctx, h.S3Config.Bucket, s3FilePath, tempFilePath, minio.PutObjectOptions{
				ContentType:     contentType,
				ContentEncoding: contentEncoding,
			})
			if err != nil {
				return err
			}
			slog.Info(fmt.Sprintf("Successfully uploaded %s of size %d", s3FilePath, info.Size))

			err = os.Remove(tempFilePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// pushFactsToLagoonApi acts as the interface between GraphQL and internal Types
func (h *Messaging) pushFactsToLagoonApi(facts []LagoonFact, resource ResourceDestination) error {

	logger := slog.With(
		"ProjectName", resource.Project,
		"EnvironmentName", resource.Environment,
	)
	apiClient := graphql.NewClient(h.LagoonAPI.Endpoint, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, h: h}})

	slog.Debug("Attempting to add facts",
		"Number", len(facts),
	)

	processedFacts := make([]lagoonclient.AddFactInput, len(facts))
	for i, fact := range facts {
		processedFacts[i] = lagoonclient.AddFactInput{
			Id:          fact.Id,
			Environment: fact.Environment,
			Name:        fact.Name,
			Value:       fact.Value,
			Source:      fact.Source,
			Description: fact.Description,
			KeyFact:     fact.KeyFact,
			Type:        lagoonclient.FactType(fact.Type),
			Category:    fact.Category,
		}

	}

	result, err := lagoonclient.AddFacts(context.TODO(), apiClient, processedFacts)
	if err != nil {
		return err
	}

	if h.EnableDebug {
		for _, fact := range facts {
			logger.Debug("Added fact", "Name", fact.Name, "Value", fact.Value)
		}
	}

	logger.Debug("Response from API",
		"result", result,
	)
	return nil
}

func decodeGzipString(encodedString string) (result interface{}, err error) {
	// base64 decode it
	base64Decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encodedString))
	decodedGzipReader, err := gzip.NewReader(base64Decoder)
	if err != nil {
		return "", err
	}

	// Decode json from reader
	var data interface{}
	jsonDecoder := json.NewDecoder(decodedGzipReader)
	err = jsonDecoder.Decode(&data)
	if err != nil && err != io.EOF {
		return "", err
	}

	return data, nil
}

// TODO: this seems to be dead code. Remove?
func scanKeyFactsFile(file string) ([]string, error) {
	var expectedKeyFacts []string

	f, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatalf("open file error: %v", err)
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if !strings.HasPrefix(string(line), "#") {
			expectedKeyFacts = append(expectedKeyFacts, sc.Text())
		}
	}
	if err := sc.Err(); err != nil {
		//TODO: Note that the pre-refactored behaviour is a FatalF, which should just exit the service completely
		//log.Fatalf("scan file error: %v", err)
		//return nil, err
		slog.Error("Scan file Error", "Error", err.Error())
		os.Exit(1)
	}
	return expectedKeyFacts, nil
}

// toLagoonInsights sends logs to the lagoon-insights message queue
func (h *Messaging) toLagoonInsights(messageQueue mq.MQ, message map[string]interface{}) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		// TODO: BETTER ERROR HANDLING
		slog.Debug("Unable to encode message as JSON", "Error", err.Error())
	}
	producer, err := messageQueue.AsyncProducer("lagoon-insights")
	if err != nil {
		// TODO: BETTER ERROR HANDLING
		slog.Debug("Failed to get async producer", "Error", err.Error())
		return
	}
	producer.Produce(msgBytes)
}
