// dns_update.go

package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ConstructNSUpdateScript constructs the nsupdate script based on the event.
// It includes the server and zone directives.
// Includes created and last_updated timestamps in the TXT meta-record for "created" and "updated" events.
func ConstructNSUpdateScript(host string, port string, zone string, fqdn string, recordType string, value string, recordID int, event string, ttl int, created string, lastUpdated string) string {
	var script strings.Builder

	// Specify the server
	script.WriteString(fmt.Sprintf("server %s %s\n", host, port))

	// Specify the zone
	script.WriteString(fmt.Sprintf("zone %s\n", zone))

	switch event {
	case "created":
		// Use provided TTL or default to 300 if TTL is zero
		if ttl <= 0 {
			ttl = 300
		}
		// Add the A record
		script.WriteString(fmt.Sprintf("update add %s %d IN %s %s\n", fqdn, ttl, recordType, value))
		// Add the TXT meta-record with record_id, created, and last_updated
		script.WriteString(fmt.Sprintf("update add %s %d IN TXT \"record_id: %d, created: %s, last_updated: %s\"\n", fqdn, ttl, recordID, created, lastUpdated))
	case "deleted":
		// Remove the A record (specific value)
		script.WriteString(fmt.Sprintf("update delete %s %s %s\n", fqdn, recordType, value))
		// Remove the TXT meta-record by matching the record_id
		script.WriteString(fmt.Sprintf("update delete %s TXT \"record_id: %d\"\n", fqdn, recordID))
	}

	// Send the update
	script.WriteString("send\n")

	return script.String()
}

// ExecuteNSUpdate executes the nsupdate command with the provided script using the TSIG key file.
func ExecuteNSUpdate(script string, config *Config) error {
	// Prepare the command with the -k flag pointing to the TSIG key file
	cmd := exec.Command("nsupdate", "-v", "-k", config.TSIGKeyFile)

	// Provide the script via stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	// Write the script to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, script)
	}()

	// Capture stderr for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("nsupdate error: %v, stderr: %s", err, stderr.String())
	}

	return nil
}
