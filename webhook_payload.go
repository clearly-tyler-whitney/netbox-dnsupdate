// webhook_payload.go

package main

// WebhookPayload represents the structure of the webhook payload.
type WebhookPayload struct {
	Event     string         `json:"event"`
	Username  string         `json:"username"`
	RequestID string         `json:"request_id"`
	Timestamp string         `json:"timestamp"`
	Data      RecordData     `json:"data"`
	Snapshots *SnapshotsData `json:"snapshots"`
}

// RecordData represents the data of the DNS record in the webhook payload.
type RecordData struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"` // Added Name field
	FQDN       string   `json:"fqdn"`
	Type       string   `json:"type"`
	Value      string   `json:"value"`
	TTL        *int     `json:"ttl"`
	DisablePTR bool     `json:"disable_ptr"`
	Zone       ZoneData `json:"zone"`
}

// Snapshot represents the state of a DNS record before or after a change.
type Snapshot struct {
	ID         int    `json:"id"`
	Name       string `json:"name"` // Added Name field
	FQDN       string `json:"fqdn"`
	Type       string `json:"type"`
	Value      string `json:"value"`
	TTL        *int   `json:"ttl"`
	DisablePTR bool   `json:"disable_ptr"`
	Zone       int    `json:"zone"` // Zone is an integer (ID)
}

// ZoneData represents DNS zone information.
type ZoneData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SnapshotsData holds the pre-change and post-change snapshots.
type SnapshotsData struct {
	PreChange  *Snapshot `json:"prechange"`
	PostChange *Snapshot `json:"postchange"`
}

// Validate ensures that the webhook payload contains the necessary data.
func (wp *WebhookPayload) Validate() error {
	// Add validation logic as needed.
	return nil
}
