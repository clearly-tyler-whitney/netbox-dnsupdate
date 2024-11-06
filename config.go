// config.go

package main

import (
	"encoding/json"
	"os"
)

// Config represents the application configuration.
type Config struct {
	ListenAddress     string `json:"listen_address"`
	BindServerAddress string `json:"bind_server_address"`
	TSIGKeyFile       string `json:"tsig_key_file"`
	LogLevel          string `json:"log_level"`
	LogFormat         string `json:"log_format"`
}

// LoadConfig loads the configuration from environment variables, a file, or defaults.
func LoadConfig() (*Config, error) {
	// Set default values
	config := &Config{
		ListenAddress:     ":8080",
		BindServerAddress: "127.0.0.1:53",
		TSIGKeyFile:       "/etc/nsupdate.key",
		LogLevel:          "info",
		LogFormat:         "logfmt",
	}

	// Override defaults with environment variables if set
	if val := os.Getenv("LISTEN_ADDRESS"); val != "" {
		config.ListenAddress = val
	}
	if val := os.Getenv("BIND_SERVER_ADDRESS"); val != "" {
		config.BindServerAddress = val
	}
	if val := os.Getenv("TSIG_KEY_FILE"); val != "" {
		config.TSIGKeyFile = val
	}
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		config.LogLevel = val
	}
	if val := os.Getenv("LOG_FORMAT"); val != "" {
		config.LogFormat = val
	}

	// Attempt to load configuration from file if it exists
	configFile := "config.json"
	if _, err := os.Stat(configFile); err == nil {
		file, err := os.Open(configFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return nil, err
		}
	}

	return config, nil
}
