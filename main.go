// main.go

package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize the RecordLockManager
	lockManager := &RecordLockManager{}

	// Register HTTP handlers
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		webhookHandler(w, r, config, lockManager)
	})

	// Health check endpoints
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/ready", readyHandler)

	log.Printf("Starting server on %s...\n", config.ListenAddress)
	if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
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
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.Printf("Error reading request body: %v\n", err)
		return
	}
	defer r.Body.Close()

	// Parse the JSON payload
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		log.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Validate the payload
	if err := payload.Validate(); err != nil {
		http.Error(w, "Invalid payload data", http.StatusBadRequest)
		log.Printf("Payload validation error: %v\n", err)
		return
	}

	// Determine the event type
	eventType := strings.ToLower(payload.Event)
	switch eventType {
	case "created":
		handleCreatedEvent(w, r, config, lockManager, &payload)
	case "deleted":
		handleDeletedEvent(w, r, config, lockManager, &payload)
	case "updated":
		handleUpdatedEvent(w, r, config, lockManager, &payload)
	default:
		http.Error(w, "Unsupported event type", http.StatusBadRequest)
		log.Printf("Unsupported event type: %s\n", payload.Event)
	}
}

// handleCreatedEvent processes "created" webhook events.
func handleCreatedEvent(w http.ResponseWriter, r *http.Request, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Extract necessary data from the payload
	fqdn := payload.Data.FQDN
	recordType := payload.Data.Type
	value := payload.Data.Value
	recordID := payload.Data.ID // Changed from payload.NetboxDNSPluginRecordID to payload.Data.ID

	// Extract TTL, default to 300 if nil or <=0
	ttl := 300
	if payload.Data.TTL != nil && *payload.Data.TTL > 0 {
		ttl = *payload.Data.TTL
	}

	// Construct the nsupdate script
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		value,
		recordID,
		"created",
		ttl,
	)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		mutex := lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn, mutex)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			log.Printf("Failed to execute nsupdate for %s: %v\n", fqdn, err)
			return
		}

		// Log success
		log.Printf("Successfully processed CREATED event for %s by user %s with request ID %s and record ID %d\n",
			fqdn, payload.Username, payload.RequestID, recordID)
	}()

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleDeletedEvent processes "deleted" webhook events.
func handleDeletedEvent(w http.ResponseWriter, r *http.Request, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Extract necessary data from the payload
	fqdn := payload.Data.FQDN
	recordType := payload.Data.Type
	value := payload.Data.Value
	recordID := payload.Data.ID // Changed from payload.NetboxDNSPluginRecordID to payload.Data.ID

	// Construct the nsupdate script
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		value,
		recordID,
		"deleted",
		0, // TTL is irrelevant for deletion
	)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		mutex := lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn, mutex)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			log.Printf("Failed to execute nsupdate for %s: %v\n", fqdn, err)
			return
		}

		// Log success
		log.Printf("Successfully processed DELETED event for %s by user %s with request ID %s and record ID %d\n",
			fqdn, payload.Username, payload.RequestID, recordID)
	}()

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleUpdatedEvent processes "updated" webhook events.
func handleUpdatedEvent(w http.ResponseWriter, r *http.Request, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Extract prechange and postchange data
	preChange := payload.Snapshots.PreChange
	postChange := payload.Snapshots.PostChange

	// Ensure that the record IDs match (assuming both snapshots pertain to the same record)
	if preChange.Name != postChange.Name || preChange.Type != postChange.Type || preChange.Zone != postChange.Zone {
		http.Error(w, "Mismatch in prechange and postchange data", http.StatusBadRequest)
		log.Printf("Mismatch in prechange and postchange data for record ID %d\n", payload.Data.ID)
		return
	}

	fqdn := postChange.FQDN
	recordType := postChange.Type
	newValue := postChange.Value
	oldValue := preChange.Value
	recordID := payload.Data.ID // Changed from payload.NetboxDNSPluginRecordID to payload.Data.ID

	// Extract TTL, default to 300 if nil or <=0
	ttl := 300
	if postChange.TTL != nil && *postChange.TTL > 0 {
		ttl = *postChange.TTL
	}

	// Construct the nsupdate scripts
	// 1. Remove the old A record and its TXT record
	removeScript := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		oldValue,
		recordID,
		"deleted",
		0, // TTL is irrelevant for deletion
	)

	// 2. Add the new A record and its TXT record
	addScript := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		payload.Data.Zone.Name,
		fqdn,
		recordType,
		newValue,
		recordID,
		"created",
		ttl,
	)

	// Start a goroutine to handle the DNS updates
	go func() {
		// Acquire lock for the FQDN
		mutex := lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn, mutex)

		// Execute nsupdate for removal
		err := ExecuteNSUpdate(removeScript, config)
		if err != nil {
			log.Printf("Failed to execute nsupdate for removal of %s: %v\n", fqdn, err)
			return
		}

		// Execute nsupdate for addition
		err = ExecuteNSUpdate(addScript, config)
		if err != nil {
			log.Printf("Failed to execute nsupdate for addition of %s: %v\n", fqdn, err)
			return
		}

		// Log success
		log.Printf("Successfully processed UPDATED event for %s by user %s with request ID %s and record ID %d\n",
			fqdn, payload.Username, payload.RequestID, recordID)
	}()

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// extractHost extracts the host from the BindServerAddress
func extractHost(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// Assume no port specified, return the whole address
		return address
	}
	return host
}

// extractPort extracts the port from the BindServerAddress or returns default "53"
func extractPort(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		// Assume no port specified, return default "53"
		return "53"
	}
	return port
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
