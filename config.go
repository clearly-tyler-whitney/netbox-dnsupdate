// config.go

package main

import (
	"fmt"
	"os"
)

// Config holds the configuration parameters for the application.
type Config struct {
	BindServerAddress string // e.g., "127.0.0.1:53"
	TSIGKeyFile       string // Path to the TSIG key file, e.g., "/etc/bind/keys/ddns-key.conf"
	ListenAddress     string // e.g., ":8080"
	LogLevel          string // e.g., "INFO"
	LogFormat         string // e.g., "json" or "logfmt"
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	config := &Config{
		BindServerAddress: getEnv("BIND_SERVER_ADDRESS", "127.0.0.1:53"),
		TSIGKeyFile:       os.Getenv("TSIG_KEY_FILE"),
		ListenAddress:     getEnv("WEBHOOK_LISTEN_ADDRESS", ":8080"),
		LogLevel:          getEnv("LOG_LEVEL", "INFO"),
		LogFormat:         getEnv("LOG_FORMAT", "logfmt"), // Default to logfmt
	}

	if config.TSIGKeyFile == "" {
		return nil, fmt.Errorf("TSIG_KEY_FILE environment variable is required")
	}

	return config, nil
}

// getEnv retrieves environment variables or returns a default value.
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
