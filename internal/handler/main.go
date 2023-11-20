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
			log.Println(err,
				fmt.Sprintf(
					"Failed to initialize message queue manager, retrying in %d seconds, attempt %d/%d",
					h.ConnectionRetryInterval,
					attempt,
					h.ConnectionAttempts,
				),
			)
			time.Sleep(time.Duration(h.ConnectionRetryInterval) * time.Second)
		}
		return attempt < h.ConnectionAttempts, err
	})
	if err != nil {
		log.Fatalf("Finally failed to initialize message queue manager: %v", err)
	}
	defer messageQueue.Close()

	go func() {
		for err := range messageQueue.Error() {
			log.Println(fmt.Sprintf("Caught error from message queue: %v", err))
		}
	}()

	forever := make(chan bool)

	// Handle any tasks that go to the queue
	log.Println("Listening for messages in queue lagoon-insights:items")
	err = messageQueue.SetConsumerHandler("items-queue", h.processMessageQueue)
	if err != nil {
		log.Println(fmt.Sprintf("Failed to set handler to consumer `%s`: %v", "items-queue", err))
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
		if t.h.EnableDebug {
			log.Println(err)
		}
		return nil, err
	}
	req.Header.Set("Authorization", "bearer "+token)
	return t.wrapped.RoundTrip(req)
}

// Incoming payload may contain facts or problems, so we need to handle these differently
func (h *Messaging) sendToLagoonAPI(incoming *InsightsMessage, resource ResourceDestination, insights InsightsData) (err error) {
	apiClient := h.getApiClient()

	if resource.Project == "" && resource.Environment == "" {
		return fmt.Errorf("no resource definition labels could be found in payload (i.e. lagoon.sh/project or lagoon.sh/environment)")
	}

	if insights.InputPayload == Payload && insights.LagoonType == Facts {
		for _, p := range incoming.Payload {
			parserFilterLoopForPayloads(insights, p, h, apiClient, resource)
		}
	}

	if insights.InputPayload == BinaryPayload && insights.LagoonType == Facts {
		for _, p := range incoming.BinaryPayload {
			parserFilterLoopForBinaryPayloads(insights, p, h, apiClient, resource)
		}
	}

	return nil
}

func parserFilterLoopForBinaryPayloads(insights InsightsData, p string, h *Messaging, apiClient graphql.Client, resource ResourceDestination) {
	for _, filter := range parserFilters {

		result, source, err := filter(h, insights, p, apiClient, resource)
		if err != nil {
			log.Println(fmt.Errorf(err.Error()))
		}

		processResultset(result, err, h, apiClient, resource, source)
	}
}

func parserFilterLoopForPayloads(insights InsightsData, p PayloadInput, h *Messaging, apiClient graphql.Client, resource ResourceDestination) {
	for _, filter := range parserFilters {
		var result []interface{}
		var source string

		json, err := json.Marshal(p)
		if err != nil {
			log.Println(fmt.Errorf(err.Error()))
		}

		result, source, err = filter(h, insights, fmt.Sprintf("%s", json), apiClient, resource)
		if err != nil {
			log.Println(fmt.Errorf(err.Error()))
		}

		processResultset(result, err, h, apiClient, resource, source)
	}
}

// processResultset will send results as facts to the lagoon api after processing via a parser filter
func processResultset(result []interface{}, err error, h *Messaging, apiClient graphql.Client, resource ResourceDestination, source string) {
	project, environment, apiErr := determineResourceFromLagoonAPI(apiClient, resource)
	if apiErr != nil {
		log.Println(apiErr)
	}

	// Even if we don't find any new facts, we need to delete the existing ones
	// since these may be the end product of a filter process
	apiErr = h.deleteExistingFactsBySource(apiClient, environment, source, project)
	if apiErr != nil {
		log.Printf("%s", apiErr.Error())
	}

	for _, r := range result {
		if fact, ok := r.(LagoonFact); ok {
			// Handle single fact
			err = h.sendFactsToLagoonAPI([]LagoonFact{fact}, apiClient, resource, source)
			if err != nil {
				fmt.Println(err)
			}
		} else if facts, ok := r.([]LagoonFact); ok {
			// Handle slice of facts
			h.sendFactsToLagoonAPI(facts, apiClient, resource, source)
		} else {
			// Unexpected type returned from filter()
			log.Printf("unexpected type returned from filter(): %T\n", r)
		}
	}
}

func (h *Messaging) sendFactsToLagoonAPI(facts []LagoonFact, apiClient graphql.Client, resource ResourceDestination, source string) error {

	if EnableDebug {
		log.Printf("[DEBUG] matched %d number of fact(s) for '%v:%v', from source '%s'", len(facts), resource.Project, resource.Environment, source)
	}

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

	log.Printf("Previous facts deleted for '%s:%s' and source '%s'", project.Name, environment.Name, source)
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
		log.Printf("Successfully created %s", h.S3Config.Bucket)
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

		log.Printf("Successfully uploaded %s of size %d", objectName, info.Size)
	}

	if len(incoming.BinaryPayload) != 0 {
		for _, p := range incoming.BinaryPayload {
			result, err := decodeGzipString(p)
			if err != nil {
				fmt.Errorf(err.Error())
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
					fmt.Errorf(err.Error())
				}
			} else {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				gz.Write(resultJson)
				gz.Close()
				err = ioutil.WriteFile(tempFilePath, buf.Bytes(), 0644)
				if err != nil {
					fmt.Errorf(err.Error())
				}
			}

			s3FilePath := strings.ToLower(fmt.Sprintf("insights/%s/%s/%s", resource.Project, resource.Environment, objectName))
			info, err := minioClient.FPutObject(ctx, h.S3Config.Bucket, s3FilePath, tempFilePath, minio.PutObjectOptions{
				ContentType:     contentType,
				ContentEncoding: contentEncoding,
			})
			if err != nil {
				fmt.Errorf(err.Error())
			}
			log.Printf("Successfully uploaded %s of size %d\n", s3FilePath, info.Size)

			err = os.Remove(tempFilePath)
			if err != nil {
				fmt.Errorf(err.Error())
			}
		}
	}

	return nil
}

// pushFactsToLagoonApi acts as the interface between GraphQL and internal Types
func (h *Messaging) pushFactsToLagoonApi(facts []LagoonFact, resource ResourceDestination) error {
	apiClient := graphql.NewClient(h.LagoonAPI.Endpoint, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, h: h}})

	if EnableDebug {
		log.Printf("[DEBUG] attempting to add %d fact(s)...", len(facts))
	}

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
			log.Println("[DEBUG]", fact.Name, ":", fact.Value)
		}
	}

	log.Println(result)
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
		log.Fatalf("scan file error: %v", err)
		return nil, err
	}
	return expectedKeyFacts, nil
}

// toLagoonInsights sends logs to the lagoon-insights message queue
func (h *Messaging) toLagoonInsights(messageQueue mq.MQ, message map[string]interface{}) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		if h.EnableDebug {
			log.Println("[DEBUG]", err, "Unable to encode message as JSON")
		}
	}
	producer, err := messageQueue.AsyncProducer("lagoon-insights")
	if err != nil {
		log.Println(fmt.Sprintf("Failed to get async producer: %v", err))
		return
	}
	producer.Produce(msgBytes)
}
