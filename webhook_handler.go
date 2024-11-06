// webhook_handler.go

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// webhookHandler handles incoming webhook POST requests.
func webhookHandler(w http.ResponseWriter, r *http.Request, config *Config, lockManager *RecordLockManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logError("Error reading request body", "err", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the incoming JSON payload if log level is DEBUG
	logDebug("Received webhook payload", "payload", string(body))

	// Parse the JSON payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		logError("Error parsing JSON", "err", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate the payload
	if err := payload.Validate(); err != nil {
		logError("Payload validation error", "err", err)
		http.Error(w, "Invalid payload data", http.StatusBadRequest)
		return
	}

	// Determine the event type
	eventType := strings.ToLower(payload.Event)
	switch eventType {
	case "created":
		handleCreatedEvent(w, config, lockManager, &payload)
	case "deleted":
		handleDeletedEvent(w, config, lockManager, &payload)
	case "updated":
		handleUpdatedEvent(w, config, lockManager, &payload)
	default:
		logError("Unsupported event type", "event", payload.Event)
		http.Error(w, "Unsupported event type", http.StatusBadRequest)
	}
}

// healthzHandler responds with "OK" for health checks.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readyHandler responds with "READY" for readiness checks.
func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}
