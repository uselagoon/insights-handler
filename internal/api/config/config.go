package config

import (
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

func LoadConfig(dbUrl, dbHost, dbUser, dbPass, dbPort string) (*Config, error) {
	fmt.Println("dbUrl:", dbUrl)
	fmt.Println("dbHost:", dbHost)
	fmt.Println("dbUser:", dbUser)
	fmt.Println("dbPass:", dbPass)
	fmt.Println("dbPort:", dbPort)

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

	fmt.Println(dbUrl)
	fmt.Println(dbUser)
	fmt.Println(dbPass)
	fmt.Println(dbHost)
	fmt.Println(dbPort)
	fmt.Println(serverPortStr)
	fmt.Println(logLevel)

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
