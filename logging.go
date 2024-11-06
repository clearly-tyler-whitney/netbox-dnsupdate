// logging.go

package main

import (
	"log"
	"strings"
)

// Log levels
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

var (
	currentLogLevel int
)

// Initialize log levels
func initLogLevel(envLevel string, flagLevel string) {
	levelMap := map[string]int{
		"DEBUG": DEBUG,
		"INFO":  INFO,
		"WARN":  WARN,
		"ERROR": ERROR,
	}

	// Default log level
	currentLogLevel = INFO

	// Check environment variable
	if lvl, exists := levelMap[strings.ToUpper(envLevel)]; exists {
		currentLogLevel = lvl
	}

	// Override with flag if provided
	if flagLevel != "" {
		if lvl, exists := levelMap[strings.ToUpper(flagLevel)]; exists {
			currentLogLevel = lvl
		} else {
			log.Printf("Unknown log level '%s', defaulting to INFO", flagLevel)
		}
	}
}

// Logging helper functions
func logDebug(format string, v ...interface{}) {
	if currentLogLevel <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func logInfo(format string, v ...interface{}) {
	if currentLogLevel <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

func logWarn(format string, v ...interface{}) {
	if currentLogLevel <= WARN {
		log.Printf("[WARN] "+format, v...)
	}
}

func logError(format string, v ...interface{}) {
	if currentLogLevel <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}
