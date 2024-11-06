// logger.go

package main

import (
	"os"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

var (
	logger   log.Logger
	logLevel level.Option
)

// initLogger initializes the global logger.
func initLogger(config *Config) {
	// Determine the log level
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		logLevel = level.AllowDebug()
	case "info":
		logLevel = level.AllowInfo()
	case "warn":
		logLevel = level.AllowWarn()
	case "error":
		logLevel = level.AllowError()
	default:
		logLevel = level.AllowInfo()
	}

	// Determine the log format
	if config.LogFormat == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	} else {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	}

	logger = level.NewFilter(logger, logLevel)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}

// logDebug logs a debug message.
func logDebug(msg string, keyvals ...interface{}) {
	level.Debug(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

// logInfo logs an info message.
func logInfo(msg string, keyvals ...interface{}) {
	level.Info(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

// logWarn logs a warning message.
func logWarn(msg string, keyvals ...interface{}) {
	level.Warn(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}

// logError logs an error message.
func logError(msg string, keyvals ...interface{}) {
	level.Error(logger).Log(append([]interface{}{"msg", msg}, keyvals...)...)
}
