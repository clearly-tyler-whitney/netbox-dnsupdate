// webhook_payload.go

package main

import (
	"fmt"
	"strings"
)

// WebhookPayload represents the structure of the incoming webhook JSON.
type WebhookPayload struct {
	Event     string     `json:"event"`      // "created", "updated", or "deleted"
	Timestamp string     `json:"timestamp"`  // e.g., "2024-11-01T16:51:42.053914+00:00"
	Model     string     `json:"model"`      // e.g., "record"
	Username  string     `json:"username"`   // Username triggering the webhook
	RequestID string     `json:"request_id"` // Unique identifier for the request
	Data      RecordData `json:"data"`       // Current state of the record
	Snapshots *Snapshots `json:"snapshots"`  // Prechange and postchange snapshots (nullable)
}

// RecordData represents the current state of the DNS record.
type RecordData struct {
	ID          int      `json:"id"` // This is the netbox_dns_plugin_record_id
	FQDN        string   `json:"fqdn"`
	Type        string   `json:"type"`
	Value       string   `json:"value"`
	Status      string   `json:"status"`
	TTL         *int     `json:"ttl"` // Optional, pointer to differentiate between zero and nil
	Zone        ZoneData `json:"zone"`
	LastUpdated string   `json:"last_updated"`
	DisablePTR  bool     `json:"disable_ptr"`
}

// Snapshots represents the prechange and postchange states.
type Snapshots struct {
	PreChange  *Snapshot `json:"prechange"`  // Nullable
	PostChange Snapshot  `json:"postchange"` // Non-null
}

// Snapshot represents a single snapshot of the DNS record.
type Snapshot struct {
	FQDN       string `json:"fqdn"`
	Type       string `json:"type"`
	Value      string `json:"value"`
	TTL        *int   `json:"ttl"` // Optional
	Zone       int    `json:"zone"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	DisablePTR bool   `json:"disable_ptr"`
}

// ZoneData represents the DNS zone information.
type ZoneData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Validate checks if the payload contains all required fields.
func (p *WebhookPayload) Validate() error {
	if p.Event == "" || p.Username == "" || p.RequestID == "" ||
		p.Data.Zone.Name == "" || p.Data.Type == "" || p.Data.FQDN == "" || p.Data.Value == "" ||
		p.Data.Status == "" {
		return fmt.Errorf("missing required fields in payload")
	}

	// Validate status field
	if p.Data.Status != "active" && p.Data.Status != "inactive" {
		return fmt.Errorf("invalid status value: %s", p.Data.Status)
	}

	// Validate event field
	event := strings.ToLower(p.Event)
	if event != "created" && event != "updated" && event != "deleted" {
		return fmt.Errorf("invalid event type: %s", p.Event)
	}

	return nil
}
