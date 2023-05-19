package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL      string
	DatabseHost      string
	DatabaseUsername string
	DatabasePassword string
	DatabasePort     string
	ServerPort       int
	LogLevel         string
}

var (
	dbUrl  string
	dbUser string
	dbPass string
	dbHost string
	dbPort string
)

func LoadConfig() (*Config, error) {
	flag.StringVar(&dbUrl, "db-url", "", "The path to connect to the PostgreSQL instance.")
	flag.StringVar(&dbHost, "db-host", "localhost", "The host of the PostgreSQL instance.")
	flag.StringVar(&dbUser, "db-username", "postgres", "The username of the PostgreSQL instance.")
	flag.StringVar(&dbPass, "db-password", "example", "The password of the PostgreSQL instance.")
	flag.StringVar(&dbPort, "db-port", "5432", "The port for the PostgreSQL host.")
	flag.Parse()

	if dbUrl := os.Getenv("DATABASE_URL"); dbUrl != "" {
		dbUrl = dbUrl
	}
	if dbUser := os.Getenv("DATABASE_USERNAME"); dbUser != "" {
		dbUser = dbUser
	}
	if dbPass := os.Getenv("DATABASE_PASSWORD"); dbPass != "" {
		dbPass = dbPass
	}
	if dbHost := os.Getenv("DATABASE_HOST"); dbHost != "" {
		dbHost = dbHost
	}
	if dbPort := os.Getenv("DATABASE_PORT"); dbPort != "" {
		dbPort = dbPort
	}

	serverPortStr := os.Getenv("SERVER_PORT")
	logLevel := os.Getenv("LOG_LEVEL")

	//default values if environment variables are not provided
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUrl == "" {
		dbUrl = fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", dbUser, dbPass, dbHost, dbPort)
	}
	if serverPortStr == "" {
		serverPortStr = "8888"
	}
	if logLevel == "" {
		logLevel = "info"
	}

	// fmt.Println(dbUrl)
	// fmt.Println(dbUser)
	// fmt.Println(dbPass)
	// fmt.Println(dbHost)
	// fmt.Println(dbPort)

	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		log.Fatalf("Invalid SERVER_PORT value: %s", err)
	}

	if err != nil {
		log.Fatalf("Invalid ENABLE_CACHING value: %s", err)
	}

	config := &Config{
		DatabaseURL: dbUrl,
		ServerPort:  serverPort,
		LogLevel:    logLevel,
	}

	return config, err
}
