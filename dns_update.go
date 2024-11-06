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
// Removed the 'zone' parameter and the zone declaration.
func ConstructNSUpdateScript(host, port, fqdn, recordType, oldValue, newValue, event string, ttl int) string {
	var script strings.Builder

	// Specify the server
	script.WriteString(fmt.Sprintf("server %s %s\n", host, port))

	// Skipping the zone declaration

	switch event {
	case "created":
		if ttl <= 0 {
			ttl = 300
		}
		script.WriteString(fmt.Sprintf("update add %s %d IN %s %s\n", fqdn, ttl, recordType, newValue))
	case "deleted":
		script.WriteString(fmt.Sprintf("update delete %s %s %s\n", fqdn, recordType, oldValue))
	case "updated":
		script.WriteString(fmt.Sprintf("update delete %s %s %s\n", fqdn, recordType, oldValue))
		script.WriteString(fmt.Sprintf("update add %s %d IN %s %s\n", fqdn, ttl, recordType, newValue))
	}
	// Send the update
	script.WriteString("send\n")

	return script.String()
}

// ConstructPTRUpdateScript constructs the nsupdate script for PTR records.
// This function also skips the 'zone' declaration.
func ConstructPTRUpdateScript(host, port, ptrName, fqdn, event string, ttl int) string {
	var script strings.Builder

	// Specify the server
	script.WriteString(fmt.Sprintf("server %s %s\n", host, port))

	// Skipping the zone declaration

	switch event {
	case "created":
		if ttl <= 0 {
			ttl = 300
		}
		script.WriteString(fmt.Sprintf("update add %s %d IN PTR %s\n", ptrName, ttl, fqdn))
	case "deleted":
		script.WriteString(fmt.Sprintf("update delete %s PTR %s\n", ptrName, fqdn))
	}
	// Send the update
	script.WriteString("send\n")

	return script.String()
}

// ExecuteNSUpdate executes the nsupdate script.
func ExecuteNSUpdate(script string, config *Config) error {
	cmd := exec.Command("nsupdate", "-k", config.TSIGKeyFile)

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
		return fmt.Errorf("nsupdate error: %v\nUPDATE SECTION:\n%s\nstderr: %s", err, script, stderr.String())
	}

	return nil
}
