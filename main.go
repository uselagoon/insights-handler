package main

import (
	"flag"
	"fmt"
	"github.com/cheshir/go-mq"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/handler"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/service"
	"log/slog"
	"os"
	"strconv"
	"time"
)

var (
	httpListenPort               = os.Getenv("HTTP_LISTEN_PORT")
	mqUser                       string
	mqPass                       string
	mqHost                       string
	mqPort                       string
	mqWorkers                    int
	rabbitReconnectRetryInterval int
	startupConnectionAttempts    int
	startupConnectionInterval    int
	lagoonAPIHost                string
	lagoonAppID                  string
	jwtTokenSigningKey           string
	jwtAudience                  string
	insightsQueueName            string
	insightsExchange             string
	jwtSubject                   string
	jwtIssuer                    string
	s3SecretAccessKey            string
	s3Origin                     string
	s3Bucket                     string
	s3Region                     string
	s3AccessKeyID                string
	filterTransformerFile        string
	s3useSSL                     bool
	disableS3Upload              bool
	disableAPIIntegration        bool
	enableDebug                  bool
	problemsFromSBOM             bool
	trivyServerEndpoint          string
	disableConsumer              bool
	disableWebservice            bool
	webservicePort               int
	webserviceListenAddress      string
)

func main() {
	flag.StringVar(&lagoonAppID, "lagoon-app-id", "insights-handler", "The appID to use that will be sent with messages.")
	flag.StringVar(&mqUser, "rabbitmq-username", "guest", "The username of the rabbitmq user.")
	flag.StringVar(&mqPass, "rabbitmq-password", "guest", "The password for the rabbitmq user.")
	flag.StringVar(&mqHost, "rabbitmq-hostname", "localhost", "The hostname for the rabbitmq host.")
	flag.StringVar(&mqPort, "rabbitmq-port", "5672", "The port for the rabbitmq host.")
	flag.IntVar(&mqWorkers, "rabbitmq-queue-workers", 1, "The number of workers to start with.")
	flag.IntVar(&rabbitReconnectRetryInterval, "rabbitmq-reconnect-retry-interval", 30, "The retry interval for rabbitmq.")
	flag.IntVar(&startupConnectionAttempts, "startup-connection-attempts", 10, "The number of startup attempts before exiting.")
	flag.IntVar(&startupConnectionInterval, "startup-connection-interval-seconds", 30, "The duration between startup attempts.")
	flag.StringVar(&lagoonAPIHost, "lagoon-api-host", "http://localhost:3000/graphql", "The host for the lagoon api.")
	flag.StringVar(&jwtTokenSigningKey, "jwt-token-signing-key", "super-secret-string", "The jwt signing token key or secret.")
	flag.StringVar(&jwtAudience, "jwt-audience", "api.dev", "The jwt audience.")
	flag.StringVar(&jwtSubject, "jwt-subject", "insights-handler", "The jwt audience.")
	flag.StringVar(&jwtIssuer, "jwt-issuer", "insights-handler", "The jwt issuer.")
	flag.StringVar(&insightsQueueName, "insights-queue-name", "lagoon-insights:items", "The name of the queue in rabbitmq to use.")
	flag.StringVar(&insightsExchange, "insights-exchange", "lagoon-insights", "The name of the exchange in rabbitmq to use.")
	flag.StringVar(&s3SecretAccessKey, "secret-access-key", "minio123", "s3 secret access key to use.")
	flag.StringVar(&s3Origin, "s3-host", "localhost:9000", "The s3 host/origin to use.")
	flag.StringVar(&s3AccessKeyID, "access-key-id", "minio", "The name of the s3Bucket to use.")
	flag.StringVar(&s3Bucket, "s3-bucket", "lagoon-insights", "The s3 bucket name.")
	flag.StringVar(&s3Region, "s3-region", "", "The s3 region.")
	flag.BoolVar(&s3useSSL, "s3-usessl", true, "Use SSL with S3")
	flag.StringVar(&filterTransformerFile, "filter-transformer-file", "./default_filter_transformers.yaml", "The filter/transformers to load.")
	flag.BoolVar(&disableS3Upload, "disable-s3-upload", false, "Disable uploading insights data to an s3 s3Bucket")
	flag.BoolVar(&disableAPIIntegration, "disable-api-integration", false, "Disable insights data integration for the Lagoon API")
	flag.BoolVar(&enableDebug, "debug", false, "Enable debugging output")
	flag.BoolVar(&problemsFromSBOM, "problems-from-sbom", false, "Pass any SBOM through Trivy")
	flag.StringVar(&trivyServerEndpoint, "trivy-server-location", "http://localhost:4954", "Trivy server endpoint")
	flag.BoolVar(&disableConsumer, "disable-broker-consumer", false, "Set this to 'true' if you'd need to run insights-handler without broker integration")
	flag.BoolVar(&disableWebservice, "disable-webservice", false, "Set to true if you need the web-service to be disabled")
	flag.IntVar(&webservicePort, "webservice-port", 3005, "Port webservice is started on")
	flag.StringVar(&webserviceListenAddress, "webservice-listen-add", "0.0.0.0", "Address on which to listen to incoming webservice connections")

	flag.Parse()

	handler.EnableDebug = enableDebug

	// get overrides from environment variables
	mqUser = getEnv("RABBITMQ_USERNAME", mqUser)
	mqPass = getEnv("RABBITMQ_PASSWORD", mqPass)
	mqHost = getEnv("RABBITMQ_ADDRESS", mqHost)
	mqPort = getEnv("RABBITMQ_PORT", mqPort)
	lagoonAPIHost = getEnv("GRAPHQL_ENDPOINT", lagoonAPIHost)
	jwtTokenSigningKey = getEnv("JWT_SECRET", jwtTokenSigningKey)
	jwtAudience = getEnv("JWT_AUDIENCE", jwtAudience)
	jwtSubject = getEnv("JWT_SUBJECT", jwtSubject)
	jwtIssuer = getEnv("JWT_ISSUER", jwtIssuer)
	insightsQueueName = getEnv("INSIGHTS_QUEUE_NAME", insightsQueueName)
	insightsExchange = getEnv("INSIGHTS_EXCHANGE", insightsExchange)
	s3Origin = getEnv("S3_FILES_HOST", s3Origin)
	s3AccessKeyID = getEnv("S3_FILES_ACCESS_KEY_ID", s3AccessKeyID)
	s3SecretAccessKey = getEnv("S3_FILES_SECRET_ACCESS_KEY", s3SecretAccessKey)
	s3Bucket = getEnv("S3_FILES_BUCKET", s3Bucket)
	s3Region = getEnv("S3_FILES_REGION", s3Region)
	filterTransformerFile = getEnv("FILTER_TRANSFORMER_FILE", filterTransformerFile)
	s3useSSL = getEnvBool("S3_USESSL", s3useSSL)
	disableAPIIntegration = getEnvBool("INSIGHTS_DISABLE_API_INTEGRATION", disableAPIIntegration)
	disableS3Upload = getEnvBool("INSIGHTS_DISABLE_S3_UPLOAD", disableS3Upload)
	problemsFromSBOM = getEnvBool("PROBLEMS_FROM_SBOM", problemsFromSBOM)
	trivyServerEndpoint = getEnv("TRIVY_SERVER_ENDPOINT", trivyServerEndpoint)
	enableDebug = getEnvBool("ENABLE_DEBUG", enableDebug)
	disableConsumer = getEnvBool("DISABLE_CONSUMER", disableConsumer)
	disableWebservice = getEnvBool("DISABLE_WEBSERVICE", disableWebservice)
	webservicePort = getEnvInt("WEBSERVICE_PORT", webservicePort)
	webserviceListenAddress = getEnv("WEBSERVICE_LISTEN_ADDRESS", webserviceListenAddress)

	// First we set up the default logger for the project

	// If we enable debugging, we set the logging level to output debug for the default logger.
	// This means we don't need to wrap debug info, simply log it at the right level
	debugLevel := slog.LevelInfo
	if enableDebug {
		debugLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: debugLevel,
	})))

	slog.Debug("problemsFromSBOM", "status", problemsFromSBOM)
	if problemsFromSBOM == true {
		if trivyServerEndpoint == "" {
			slog.Error("NO TRIVY SERVER ENDPOINT SET - exiting")
			os.Exit(1)
		}
	}

	// We begin by setting up the handler's broker connection
	broker := handler.RabbitBroker{
		Hostname:     fmt.Sprintf("%s:%s", mqHost, mqPort),
		Username:     mqUser,
		Password:     mqPass,
		QueueName:    insightsQueueName,
		ExchangeName: insightsExchange,
	}

	// graphQLConfig details how we connect to the Lagoon API
	graphQLConfig := handler.LagoonAPI{
		Endpoint:        lagoonAPIHost,
		TokenSigningKey: jwtTokenSigningKey,
		JWTAudience:     jwtAudience,
		JWTSubject:      jwtSubject,
		JWTIssuer:       jwtIssuer,
		Disabled:        disableAPIIntegration,
	}

	// s3Config details how we connect to the s3 buckets - these are used to upload files
	s3Config := handler.S3{
		SecretAccessKey: s3SecretAccessKey,
		S3Origin:        s3Origin,
		AccessKeyId:     s3AccessKeyID,
		Bucket:          s3Bucket,
		Region:          s3Region,
		UseSSL:          s3useSSL,
		Disabled:        disableS3Upload,
	}

	slog.Debug("disableS3Upload", "status", disableS3Upload)

	// Here we look at the filter json/yaml and attempt to load up the filter descriptions
	err := handler.RegisterFiltersFromDisk(filterTransformerFile)
	if err != nil {
		slog.Error("Unable to register filters from disk", "Error", err)
		os.Exit(1)
	}

	config := mq.Config{
		ReconnectDelay: time.Duration(rabbitReconnectRetryInterval) * time.Second,
		Exchanges: mq.Exchanges{
			{
				Name: "lagoon-insights",
				Type: "direct",
				Options: mq.Options{
					"durable":       true,
					"delivery_mode": "2",
					"headers":       "",
					"content_type":  "",
				},
			},
		},
		Consumers: mq.Consumers{
			{
				Name:    "items-queue",
				Queue:   "lagoon-insights:items",
				Workers: mqWorkers,
				Options: mq.Options{
					"durable":       true,
					"delivery_mode": "2",
					"headers":       "",
					"content_type":  "",
				},
			},
		},
		Queues: mq.Queues{
			{
				Name:     "lagoon-insights:items",
				Exchange: "lagoon-insights",
				Options: mq.Options{
					"durable":       true,
					"delivery_mode": "2",
					"headers":       "",
					"content_type":  "",
				},
			},
		},
		DSN: fmt.Sprintf("amqp://%s:%s@%s/", broker.Username, broker.Password, broker.Hostname),
	}

	db, err := service.SetUpDatabase(service.Dboptions{Filename: "database.sql"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Start up the web service if we need it
	if !disableWebservice {

		r, err := service.SetupRouter(db)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		go r.Run(fmt.Sprintf("%v:%v", webserviceListenAddress, webservicePort))
	}

	// Start up the consumer if we need it
	if !disableConsumer {
		messaging := handler.NewMessaging(config,
			graphQLConfig,
			s3Config,
			startupConnectionAttempts,
			startupConnectionInterval,
			enableDebug,
			problemsFromSBOM,
			trivyServerEndpoint,
			db,
		)

		// start the consumer
		//slog.Info("insights-handler is started-up")
		messaging.Consumer()
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// accepts fallback values 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
// anything else is false.
func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		rVal, _ := strconv.ParseBool(value)
		return rVal
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		rVal, _ := strconv.ParseInt(value, 10, 16)
		return int(rVal)
	}
	return fallback
}

//go:generate go run github.com/Khan/genqlient internal/lagoonclient/genqlient.yaml
