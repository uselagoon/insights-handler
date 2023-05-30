package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/cheshir/go-mq"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/uselagoon/lagoon/services/insights-handler/internal/handler"
	"github.com/uselagoon/lagoon/services/insights-handler/migrations"

	config "github.com/uselagoon/lagoon/services/insights-handler/internal/api/config"
	db "github.com/uselagoon/lagoon/services/insights-handler/internal/api/database"
	routes "github.com/uselagoon/lagoon/services/insights-handler/internal/api/routes"
)

var (
	httpListenPort                        = os.Getenv("HTTP_LISTEN_PORT")
	mqUser                                string
	mqPass                                string
	mqHost                                string
	mqPort                                string
	mqWorkers                             int
	rabbitReconnectRetryInterval          int
	startupConnectionAttempts             int
	startupConnectionInterval             int
	lagoonAPIHost                         string
	lagoonAppID                           string
	jwtTokenSigningKey                    string
	jwtAudience                           string
	insightsQueueName                     string
	insightsExchange                      string
	jwtSubject                            string
	jwtIssuer                             string
	s3SecretAccessKey                     string
	s3Origin                              string
	s3Bucket                              string
	s3Region                              string
	s3AccessKeyID                         string
	filterTransformerFile                 string
	s3useSSL                              bool
	disableS3Upload                       bool
	disableAPIIntegration                 bool
	enableDebug                           bool
	dbUrl, dbHost, dbUser, dbPass, dbPort string
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
	flag.StringVar(&jwtSubject, "jwt-subject", "actions-handler", "The jwt audience.")
	flag.StringVar(&jwtIssuer, "jwt-issuer", "actions-handler", "The jwt audience.")
	flag.StringVar(&insightsQueueName, "insights-queue-name", "lagoon-insights:items", "The name of the queue in rabbitmq to use.")
	flag.StringVar(&insightsExchange, "insights-exchange", "lagoon-insights", "The name of the exchange in rabbitmq to use.")
	flag.StringVar(&s3SecretAccessKey, "secret-access-key", "minio123", "s3 secret access key to use.")
	flag.StringVar(&s3Origin, "s3-host", "localhost:9000", "The s3 host/origin to use.")
	flag.StringVar(&s3AccessKeyID, "access-key-id", "minio", "The name of the s3Bucket to use.")
	flag.StringVar(&s3Bucket, "s3-bucket", "lagoon-insights", "The s3 bucket name.")
	flag.StringVar(&s3Region, "s3-region", "", "The s3 region.")
	flag.BoolVar(&s3useSSL, "s3-usessl", true, "Use SSL with S3")

	flag.StringVar(&filterTransformerFile, "filter-transformer-file", "./default_filter_transformers.json", "The filter/transformers to load.")
	flag.BoolVar(&disableS3Upload, "disable-s3-upload", false, "Disable uploading insights data to an s3 s3Bucket")
	flag.BoolVar(&disableAPIIntegration, "disable-api-integration", false, "Disable insights data integration for the Lagoon API")
	flag.BoolVar(&enableDebug, "debug", false, "Enable debugging output")

	flag.StringVar(&dbUrl, "db-url", dbUrl, "The path to connect to the PostgreSQL instance.")
	flag.StringVar(&dbHost, "db-host", dbHost, "The host of the PostgreSQL instance.")
	flag.StringVar(&dbUser, "db-username", dbUser, "The username of the PostgreSQL instance.")
	flag.StringVar(&dbPass, "db-password", dbPass, "The password of the PostgreSQL instance.")
	flag.StringVar(&dbPort, "db-port", dbPort, "The port for the PostgreSQL host.")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: main.go flag args [server|handler|migrate]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// access the non-flag arguments [server|handler|migrate]
	arg := args[0]

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

	dbUrl = getEnv("DATABASE_URL", dbUrl)
	dbHost = getEnv("DATABASE_HOST", dbHost)
	dbUser = getEnv("DATABASE_USERNAME", dbUser)
	dbPass = getEnv("DATABASE_PASSWORD", dbPass)
	dbPort = getEnv("DATABASE_PORT", dbPort)

	// seperate handler and server via cli args
	command := arg
	switch command {
	case "handler":
		startHandler()
		// wait indefinitely to keep the application running
		select {}
	case "server":
		startServer(dbUrl, dbHost, dbUser, dbPass, dbPort)
		// wait indefinitely to keep the application running
		select {}
	case "migrate":
		err := migrations.RunMigrations()
		if err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		fmt.Println("Migrations executed successfully")
	case "seed":
		err := migrations.RunSeed()
		if err != nil {
			log.Fatalf("Failed to run seeds: %v", err)
		}

		fmt.Println("Seeds executed successfully")
	default:
		log.Fatal("Invalid argument. Supported arguments are 'handler' and 'server'")
	}

}

func startServer(dbUrl, dbHost, dbUser, dbPass, dbPort string) {
	// setup REST api
	cfg, err := config.LoadConfig(dbUrl, dbHost, dbUser, dbPass, dbPort)
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}
	// initialise Gin router
	router := gin.Default()

	// establish db connection
	db, err := db.NewDBConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// register routes
	routes.RegisterRoutes(router, db)

	// start the server in a separate goroutine
	addr := fmt.Sprintf(":%s", fmt.Sprint(cfg.ServerPort))
	go func() {
		fmt.Printf("Server listening on %s\n", addr)
		err := router.Run(addr)
		if err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
}

func startHandler() {
	// configure the backup handler settings
	broker := handler.RabbitBroker{
		Hostname:     fmt.Sprintf("%s:%s", mqHost, mqPort),
		Username:     mqUser,
		Password:     mqPass,
		QueueName:    insightsQueueName,
		ExchangeName: insightsExchange,
	}
	graphQLConfig := handler.LagoonAPI{
		Endpoint:        lagoonAPIHost,
		TokenSigningKey: jwtTokenSigningKey,
		JWTAudience:     jwtAudience,
		JWTSubject:      jwtSubject,
		JWTIssuer:       jwtIssuer,
		Disabled:        disableAPIIntegration,
	}
	s3Config := handler.S3{
		SecretAccessKey: s3SecretAccessKey,
		S3Origin:        s3Origin,
		AccessKeyId:     s3AccessKeyID,
		Bucket:          s3Bucket,
		Region:          s3Region,
		UseSSL:          s3useSSL,
		Disabled:        disableS3Upload,
	}

	log.Println("Registering Fact Filters/Transformer")
	err := handler.RegisterFiltersFromJson(filterTransformerFile)
	if err != nil {
		log.Println(err)
	}

	log.Println("insights-handler running...")

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

	messaging := handler.NewMessaging(config,
		graphQLConfig,
		s3Config,
		startupConnectionAttempts,
		startupConnectionInterval,
		enableDebug,
	)

	// start the consumer in a separate goroutine
	go messaging.Consumer()
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

//go:generate go run github.com/Khan/genqlient internal/lagoonclient/genqlient.yaml
