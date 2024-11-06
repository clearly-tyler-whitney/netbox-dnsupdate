// main.go

package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	// Define command-line flag for log level
	logLevelFlag := flag.String("log-level", "", "Set the logging level (DEBUG, INFO, WARN, ERROR)")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("[ERROR] Configuration error: %v", err)
	}

	// Initialize log level
	initLogLevel(config.LogLevel, *logLevelFlag)

	// Initialize the RecordLockManager
	lockManager := &RecordLockManager{}

	// Register HTTP handlers
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler(w, r, config, lockManager)
	})

	// Health check endpoints
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/ready", readyHandler)

	logInfo("Starting server on %s...", config.ListenAddress)
	if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
		log.Fatalf("[ERROR] Server failed to start: %v", err)
	}
}

// webhookHandler handles incoming webhook POST requests.
func webhookHandler(w http.ResponseWriter, r *http.Request, config *Config, lockManager *RecordLockManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logError("Error reading request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse the JSON payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		logError("Error parsing JSON: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate the payload
	if err := payload.Validate(); err != nil {
		logError("Payload validation error: %v", err)
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
		logError("Unsupported event type: %s", payload.Event)
		http.Error(w, "Unsupported event type", http.StatusBadRequest)
	}
}

// handleCreatedEvent processes "created" webhook events.
func handleCreatedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	fqdn := payload.Data.FQDN
	recordType := payload.Data.Type
	value := payload.Data.Value

	// Extract TTL, default to 300 if nil or <=0
	ttl := 300
	if payload.Data.TTL != nil && *payload.Data.TTL > 0 {
		ttl = *payload.Data.TTL
	}

	// Construct the nsupdate script to add the record
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		"",    // No old value
		value, // New value
		"created",
		ttl,
	)

	logDebug("Outgoing nsupdate script for CREATED event:\n%s", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate for %s: %v", fqdn, err)
			return
		}

		// Log success
		logInfo("Successfully processed CREATED event for %s by user %s with request ID %s and record ID %d",
			fqdn, payload.Username, payload.RequestID, payload.Data.ID)
	}()

	// Handle PTR records if needed
	if !payload.Data.DisablePTR {
		handlePTRUpdate("created", nil, &payload.Data, config, lockManager)
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleDeletedEvent processes "deleted" webhook events.
func handleDeletedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	fqdn := payload.Data.FQDN
	recordType := payload.Data.Type
	value := payload.Data.Value

	// Construct the nsupdate script to delete the record
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		value, // Old value
		"",    // No new value
		"deleted",
		0, // TTL is irrelevant for deletion
	)

	logDebug("Outgoing nsupdate script for DELETED event:\n%s", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate for %s: %v", fqdn, err)
			return
		}

		// Log success
		logInfo("Successfully processed DELETED event for %s by user %s with request ID %s and record ID %d",
			fqdn, payload.Username, payload.RequestID, payload.Data.ID)
	}()

	// Handle PTR records if needed
	if !payload.Data.DisablePTR {
		handlePTRUpdate("deleted", &payload.Data, nil, config, lockManager)
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleUpdatedEvent processes "updated" webhook events.
func handleUpdatedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Check if Snapshots or PostChange is missing
	if payload.Snapshots == nil || payload.Snapshots.PostChange.FQDN == "" {
		logError("Snapshots missing or incomplete in payload for record ID %d", payload.Data.ID)
		http.Error(w, "Snapshots missing in payload", http.StatusBadRequest)
		return
	}

	preChange := payload.Snapshots.PreChange
	postChange := payload.Snapshots.PostChange

	fqdn := postChange.FQDN
	recordType := postChange.Type
	newValue := postChange.Value

	// Extract TTL, default to 300 if nil or <=0
	ttl := 300
	if postChange.TTL != nil && *postChange.TTL > 0 {
		ttl = *postChange.TTL
	}

	// Determine old value
	oldValue := ""
	if preChange != nil {
		oldValue = preChange.Value
	}

	// Construct the nsupdate script
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		oldValue,
		newValue,
		"updated",
		ttl,
	)

	logDebug("Outgoing nsupdate script for UPDATED event:\n%s", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate for %s: %v", fqdn, err)
			return
		}

		// Log success
		logInfo("Successfully processed UPDATED event for %s by user %s with request ID %s and record ID %d",
			fqdn, payload.Username, payload.RequestID, payload.Data.ID)
	}()

	// Convert snapshots to RecordData
	preData := snapshotToRecordData(preChange)
	postData := &payload.Data

	// Handle PTR records if needed
	preDisablePTR := false
	postDisablePTR := postData.DisablePTR

	if preData != nil {
		preDisablePTR = preData.DisablePTR
	}

	// Handle PTR updates accordingly
	if preDisablePTR != postDisablePTR || !postDisablePTR {
		handlePTRUpdate("updated", preData, postData, config, lockManager)
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handlePTRUpdate manages PTR records based on the event.
func handlePTRUpdate(event string, preData *RecordData, postData *RecordData, config *Config, lockManager *RecordLockManager) {
	go func() {
		var oldIP, newIP string

		if preData != nil {
			oldIP = preData.Value
		}

		if postData != nil {
			newIP = postData.Value
		}

		// Determine the PTR record names
		var oldPTRName, newPTRName string

		if oldIP != "" {
			oldPTRName = reverseDNSName(oldIP)
		}

		if newIP != "" {
			newPTRName = reverseDNSName(newIP)
		}

		// Lock on the PTR names to prevent race conditions
		if oldPTRName != "" {
			lockManager.AcquireLock(oldPTRName)
			defer lockManager.ReleaseLock(oldPTRName)
		}

		if newPTRName != "" && newPTRName != oldPTRName {
			lockManager.AcquireLock(newPTRName)
			defer lockManager.ReleaseLock(newPTRName)
		}

		// Construct nsupdate script based on the event
		var script string

		switch event {
		case "created":
			// Add PTR record
			script = ConstructPTRUpdateScript(
				extractHost(config.BindServerAddress),
				extractPort(config.BindServerAddress),
				newPTRName,
				postData.FQDN,
				"created",
				300, // Default TTL or use postData.TTL if available
			)
		case "deleted":
			// Delete PTR record
			script = ConstructPTRUpdateScript(
				extractHost(config.BindServerAddress),
				extractPort(config.BindServerAddress),
				oldPTRName,
				preData.FQDN,
				"deleted",
				0,
			)
		case "updated":
			// Delete old PTR and add new PTR
			if oldPTRName != "" && preData != nil {
				deleteScript := ConstructPTRUpdateScript(
					extractHost(config.BindServerAddress),
					extractPort(config.BindServerAddress),
					oldPTRName,
					preData.FQDN,
					"deleted",
					0,
				)
				script += deleteScript
			}
			if newPTRName != "" && postData != nil {
				createScript := ConstructPTRUpdateScript(
					extractHost(config.BindServerAddress),
					extractPort(config.BindServerAddress),
					newPTRName,
					postData.FQDN,
					"created",
					300, // Default TTL or use postData.TTL if available
				)
				script += createScript
			}
		}

		logDebug("Outgoing nsupdate script for PTR %s event:\n%s", event, script)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate for PTR record: %v", err)
			return
		}

		logInfo("Successfully processed PTR %s event for IP %s", event, newIP)
	}()
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
