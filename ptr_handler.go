// ptr_handler.go

package main

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

		// Validate IP addresses
		oldIPValid := isValidIP(oldIP)
		newIPValid := isValidIP(newIP)

		// If neither old nor new IP is valid, skip PTR update
		if !oldIPValid && !newIPValid {
			logWarn("Skipping PTR update due to invalid IP addresses",
				"event", event,
				"old_ip", oldIP,
				"new_ip", newIP,
			)
			return
		}

		// Determine the PTR record names
		var oldPTRName, newPTRName string

		if oldIPValid {
			oldPTRName = reverseDNSName(oldIP)
		}

		if newIPValid {
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
			if postData != nil {
				script = ConstructPTRUpdateScript(
					extractHost(config.BindServerAddress),
					extractPort(config.BindServerAddress),
					newPTRName,
					postData.FQDN,
					"created",
					300, // Default TTL or use postData.TTL if available
				)
			} else {
				logWarn("postData is nil in handlePTRUpdate for created event",
					"event", event,
					"new_ip", newIP,
					"new_ptr", newPTRName,
				)
				return
			}
		case "deleted":
			// Delete PTR record
			if preData != nil {
				script = ConstructPTRUpdateScript(
					extractHost(config.BindServerAddress),
					extractPort(config.BindServerAddress),
					oldPTRName,
					preData.FQDN,
					"deleted",
					0,
				)
			} else {
				logWarn("preData is nil in handlePTRUpdate for deleted event",
					"event", event,
					"old_ip", oldIP,
					"old_ptr", oldPTRName,
				)
				return
			}
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

		// If script is empty, no valid PTR updates are needed
		if script == "" {
			logWarn("No valid PTR updates needed",
				"event", event,
				"old_ip", oldIP,
				"new_ip", newIP,
			)
			return
		}

		logDebug("Outgoing nsupdate script for PTR event",
			"event", event,
			"script", script,
		)

		// Execute nsupdate
		err := ExecuteNSUpdate(script, config)
		if err != nil {
			logError("Failed to execute nsupdate for PTR record",
				"event", event,
				"err", err,
				"old_ip", oldIP,
				"new_ip", newIP,
				"old_ptr", oldPTRName,
				"new_ptr", newPTRName,
			)
			return
		}

		logInfo("Processed PTR record",
			"event", event,
			"fqdn", getFQDN(preData, postData),
			"old_ip", oldIP,
			"new_ip", newIP,
			"old_ptr", oldPTRName,
			"new_ptr", newPTRName,
		)
	}()
}
