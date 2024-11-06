// logging.go

package main

import (
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// Log levels
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

var (
	logger          log.Logger
	currentLogLevel int
)

// Initialize log levels and logger
func initLogger(config *Config, logLevelFlag string) {
	levelMap := map[string]int{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARN,
		"ERROR": ERROR,
	}

	// Default log level
	currentLogLevel = INFO

	// Check environment variable
	if lvl, exists := levelMap[strings.ToUpper(config.LogLevel)]; exists {
		currentLogLevel = lvl
	}

	// Override with flag if provided
	if logLevelFlag != "" {
		if lvl, exists := levelMap[strings.ToUpper(logLevelFlag)]; exists {
			currentLogLevel = lvl
		} else {
			logError("Unknown log level '%s', defaulting to INFO", logLevelFlag)
		}
	}

	// Initialize the logger based on the selected format
	var logFormat string
	if config.LogFormat != "" {
		logFormat = config.LogFormat
	} else {
		logFormat = os.Getenv("LOG_FORMAT")
	}

	switch strings.ToLower(logFormat) {
	case "json":
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	default:
		// Default to logfmt
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	}

	// Add timestamp to all log messages
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

// Logging helper functions
func logDebug(msg string, keyvals ...interface{}) {
	if currentLogLevel <= DEBUG {
		_ = level.Debug(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
	}
}

func logInfo(msg string, keyvals ...interface{}) {
	if currentLogLevel <= INFO {
		_ = level.Info(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
	}
}

func logWarn(msg string, keyvals ...interface{}) {
	if currentLogLevel <= WARN {
		_ = level.Warn(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
	}
}

func logError(msg string, keyvals ...interface{}) {
	if currentLogLevel <= ERROR {
		_ = level.Error(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
	}
}
