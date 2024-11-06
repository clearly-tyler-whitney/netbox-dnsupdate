// event_handlers.go

package main

import (
	"net/http"
	"strings"
)

// handleCreatedEvent processes "created" webhook events.
func handleCreatedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	fqdn := payload.Data.FQDN
	recordType := strings.ToUpper(payload.Data.Type)
	value := payload.Data.Value

	// Adjust value if record type is CNAME
	if recordType == "CNAME" {
		value = adjustCNAMEValue(value, fqdn, payload.Data.Name)
	}

	// Extract TTL, default to 300 if nil or <=0
	ttl := 300
	if payload.Data.TTL != nil && *payload.Data.TTL > 0 {
		ttl = *payload.Data.TTL
	}

	// Construct the nsupdate script to add the record
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		fqdn,
		recordType,
		"",    // No old value
		value, // New value
		"created",
		ttl,
	)

	logDebug("Outgoing nsupdate script for CREATED event", "script", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate",
				"fqdn", fqdn,
				"err", err,
				"event", "created",
				"user", payload.Username,
				"request_id", payload.RequestID,
				"record_id", payload.Data.ID,
			)
			return
		}

		// Log success
		logInfo("Processed DNS record",
			"event", "created",
			"fqdn", fqdn,
			"record_type", recordType,
			"value", value,
			"ttl", ttl,
			"user", payload.Username,
			"request_id", payload.RequestID,
			"record_id", payload.Data.ID,
		)
	}()

	// Handle PTR records if needed and recordType is A or AAAA
	if !payload.Data.DisablePTR && (recordType == "A" || recordType == "AAAA") {
		handlePTRUpdate("created", nil, &payload.Data, config, lockManager)
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleDeletedEvent processes "deleted" webhook events.
func handleDeletedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Check if Snapshots or PreChange is missing
	if payload.Snapshots == nil || payload.Snapshots.PreChange == nil {
		logError("Snapshots missing or incomplete in payload",
			"record_id", payload.Data.ID,
		)
		http.Error(w, "Snapshots missing in payload", http.StatusBadRequest)
		return
	}

	preChange := payload.Snapshots.PreChange

	fqdn := preChange.FQDN
	recordType := strings.ToUpper(preChange.Type)
	value := preChange.Value

	// Adjust value if record type is CNAME
	if recordType == "CNAME" {
		value = adjustCNAMEValue(value, fqdn, preChange.Name)
	}

	// Construct the nsupdate script to delete the record
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		fqdn,
		recordType,
		value, // Old value
		"",    // No new value
		"deleted",
		0, // TTL is irrelevant for deletion
	)

	logDebug("Outgoing nsupdate script for DELETED event", "script", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate",
				"fqdn", fqdn,
				"err", err,
				"event", "deleted",
				"user", payload.Username,
				"request_id", payload.RequestID,
				"record_id", preChange.ID,
			)
			return
		}

		// Log success
		logInfo("Processed DNS record",
			"event", "deleted",
			"fqdn", fqdn,
			"record_type", recordType,
			"value", value,
			"ttl", 0,
			"user", payload.Username,
			"request_id", payload.RequestID,
			"record_id", preChange.ID,
		)
	}()

	// Handle PTR records if needed and recordType is A or AAAA
	if !preChange.DisablePTR && (recordType == "A" || recordType == "AAAA") {
		preData := snapshotToRecordData(preChange)
		handlePTRUpdate("deleted", preData, nil, config, lockManager)
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}

// handleUpdatedEvent processes "updated" webhook events.
func handleUpdatedEvent(w http.ResponseWriter, config *Config, lockManager *RecordLockManager, payload *WebhookPayload) {
	// Check if Snapshots or PostChange is missing
	if payload.Snapshots == nil || payload.Snapshots.PostChange == nil || payload.Snapshots.PostChange.FQDN == "" {
		logError("Snapshots missing or incomplete in payload",
			"record_id", payload.Data.ID,
		)
		http.Error(w, "Snapshots missing in payload", http.StatusBadRequest)
		return
	}

	preChange := payload.Snapshots.PreChange
	postChange := payload.Snapshots.PostChange

	fqdn := postChange.FQDN
	recordType := strings.ToUpper(postChange.Type)
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

	// Adjust values if record type is CNAME
	if recordType == "CNAME" {
		newValue = adjustCNAMEValue(newValue, fqdn, postChange.Name)
		if oldValue != "" && preChange != nil {
			oldFQDN := preChange.FQDN
			oldValue = adjustCNAMEValue(oldValue, oldFQDN, preChange.Name)
		}
	}

	// Construct the nsupdate script
	script := ConstructNSUpdateScript(
		extractHost(config.BindServerAddress),
		extractPort(config.BindServerAddress),
		fqdn,
		recordType,
		oldValue,
		newValue,
		"updated",
		ttl,
	)

	logDebug("Outgoing nsupdate script for UPDATED event", "script", script)

	// Start a goroutine to handle the DNS update
	go func() {
		// Acquire lock for the FQDN
		lockManager.AcquireLock(fqdn)
		defer lockManager.ReleaseLock(fqdn)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate",
				"fqdn", fqdn,
				"err", err,
				"event", "updated",
				"user", payload.Username,
				"request_id", payload.RequestID,
				"record_id", payload.Data.ID,
			)
			return
		}

		// Log success
		logInfo("Processed DNS record",
			"event", "updated",
			"fqdn", fqdn,
			"record_type", recordType,
			"old_value", oldValue,
			"new_value", newValue,
			"ttl", ttl,
			"user", payload.Username,
			"request_id", payload.RequestID,
			"record_id", payload.Data.ID,
		)
	}()

	// Convert snapshots to RecordData
	preData := snapshotToRecordData(preChange)
	postData := snapshotToRecordData(postChange)

	// Handle PTR records if needed and recordType is A or AAAA
	if recordType == "A" || recordType == "AAAA" {
		preDisablePTR := false
		postDisablePTR := postData.DisablePTR

		if preData != nil {
			preDisablePTR = preData.DisablePTR
		}

		// Handle PTR updates accordingly
		if preDisablePTR != postDisablePTR || !postDisablePTR {
			handlePTRUpdate("updated", preData, postData, config, lockManager)
		}
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received and is being processed"))
}
