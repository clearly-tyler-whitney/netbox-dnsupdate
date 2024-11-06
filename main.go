// main.go

package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	// Define command-line flags
	logLevelFlag := flag.String("log-level", "", "Set the logging level (DEBUG, INFO, WARN, ERROR)")
	logFormatFlag := flag.String("log-format", "", "Set the logging format (json, logfmt)")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		// Use standard log for fatal errors during initialization
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		level.Error(logger).Log("msg", "Configuration error", "err", err)
		os.Exit(1)
	}

	// Override config with command-line flags if provided
	if *logLevelFlag != "" {
		config.LogLevel = *logLevelFlag
	}
	if *logFormatFlag != "" {
		config.LogFormat = *logFormatFlag
	}

	// Initialize logger
	initLogger(config)

	// Initialize the RecordLockManager
	lockManager := &RecordLockManager{}

	// Register HTTP handlers
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler(w, r, config, lockManager)
	})

	// Health check endpoints
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/ready", readyHandler)

	logInfo("Starting server", "address", config.ListenAddress)
	if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
		logError("Server failed to start", "err", err)
		os.Exit(1)
	}
}
