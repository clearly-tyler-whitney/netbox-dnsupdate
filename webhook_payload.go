// webhook_payload.go

package main

import (
	"fmt"
)

// WebhookPayload represents the structure of the incoming webhook JSON.
type WebhookPayload struct {
	Event     string     `json:"event"`      // "created", "updated", or "deleted"
	Timestamp string     `json:"timestamp"`  // e.g., "2024-11-01T16:51:42.053914+00:00"
	Model     string     `json:"model"`      // e.g., "record"
	Username  string     `json:"username"`   // Username triggering the webhook
	RequestID string     `json:"request_id"` // Unique identifier for the request
	Data      RecordData `json:"data"`       // Current state of the record
	Snapshots Snapshots  `json:"snapshots"`  // Prechange and postchange snapshots
}

// RecordData represents the current state of the DNS record.
type RecordData struct {
	ID            int                    `json:"id"` // This is the netbox_dns_plugin_record_id
	URL           string                 `json:"url"`
	Zone          ZoneData               `json:"zone"`
	Display       string                 `json:"display"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	FQDN          string                 `json:"fqdn"`
	Value         string                 `json:"value"`
	Status        string                 `json:"status"`
	TTL           *int                   `json:"ttl"` // Optional, pointer to differentiate between zero and nil
	Description   string                 `json:"description"`
	Tags          []string               `json:"tags"`
	Created       string                 `json:"created"`
	LastUpdated   string                 `json:"last_updated"`
	Managed       bool                   `json:"managed"`
	DisablePTR    bool                   `json:"disable_ptr"`
	PTRRecord     interface{}            `json:"ptr_record"`
	AddressRecord interface{}            `json:"address_record"`
	Active        bool                   `json:"active"`
	CustomFields  map[string]interface{} `json:"custom_fields"`
	Tenant        interface{}            `json:"tenant"`
	IPAMIPAddress interface{}            `json:"ipam_ip_address"`
}

// Snapshots represents the prechange and postchange states.
type Snapshots struct {
	PreChange  Snapshot `json:"prechange"`
	PostChange Snapshot `json:"postchange"`
}

// Snapshot represents a single snapshot of the DNS record.
type Snapshot struct {
	Created            string                 `json:"created"`
	LastUpdated        string                 `json:"last_updated"`
	Name               string                 `json:"name"`
	Zone               int                    `json:"zone"`
	FQDN               string                 `json:"fqdn"`
	Type               string                 `json:"type"`
	Value              string                 `json:"value"`
	Status             string                 `json:"status"`
	TTL                *int                   `json:"ttl"` // Optional
	Managed            bool                   `json:"managed"`
	PTRRecord          interface{}            `json:"ptr_record"`
	DisablePTR         bool                   `json:"disable_ptr"`
	Description        string                 `json:"description"`
	Tenant             interface{}            `json:"tenant"`
	IPAddress          string                 `json:"ip_address"`
	IPAMIPAddress      interface{}            `json:"ipam_ip_address"`
	RFC2317CNAMERecord interface{}            `json:"rfc2317_cname_record"`
	CustomFields       map[string]interface{} `json:"custom_fields"`
	Tags               []string               `json:"tags"`
}

// ZoneData represents the DNS zone information.
type ZoneData struct {
	ID            int      `json:"id"`
	URL           string   `json:"url"`
	Display       string   `json:"display"`
	Name          string   `json:"name"`
	View          ViewData `json:"view"`
	Status        string   `json:"status"`
	Active        bool     `json:"active"`
	RFC2317Prefix *string  `json:"rfc2317_prefix"`
}

// ViewData represents the DNS view information.
type ViewData struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	DefaultView bool   `json:"default_view"`
	Description string `json:"description"`
}

// Validate checks if the payload contains all required fields.
func (p *WebhookPayload) Validate() error {
	if p.Event == "" || p.Username == "" || p.RequestID == "" ||
		p.Data.Zone.Name == "" || p.Data.Type == "" || p.Data.Name == "" || p.Data.Value == "" ||
		p.Data.Status == "" {
		return fmt.Errorf("missing required fields in payload")
	}

	// Validate status field
	if p.Data.Status != "active" && p.Data.Status != "inactive" {
		return fmt.Errorf("invalid status value: %s", p.Data.Status)
	}

	// Validate event field
	if p.Event != "created" && p.Event != "updated" && p.Event != "deleted" {
		return fmt.Errorf("invalid event type: %s", p.Event)
	}

	return nil
}
