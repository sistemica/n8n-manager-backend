package n8n

import (
	"time"
)

// Instance represents an n8n instance with its configuration
type Instance struct {
	Id              string `json:"id"`
	Host            string `json:"host"`
	APIKey          string `json:"api_key"`
	IgnoreSSLErrors bool   `json:"ignore_ssl_errors"`
	CheckInterval   int    `json:"check_interval_mins"`
}

// NewInstance creates a new n8n instance
func NewInstance(id, host, apiKey string) *Instance {
	return &Instance{
		Id:            id,
		Host:          host,
		APIKey:        apiKey,
		CheckInterval: 5, // Default to 5 minutes if not specified
	}
}

// Workflow represents an n8n workflow
type Workflow struct {
	ID         string    `json:"-"`
	Name       string    `json:"name"`
	WorkflowID string    `json:"id"` // The workflow ID from n8n
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Nodes      []Node    `json:"nodes"`
	// Reference to parent instance - not serialized to JSON
	InstanceID string `json:"-"`
}

// Node represents a node in an n8n workflow
type Node struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Type        string                    `json:"type"`
	Parameters  NodeParameters            `json:"parameters"`
	Credentials map[string]NodeCredential `json:"credentials"`
	WebhookID   string                    `json:"webhookId"`
	Notes       string                    `json:"notes"`
}

// NodeParameters contains the configuration for a node
type NodeParameters struct {
	HTTPMethod     string                 `json:"httpMethod"`
	Path           string                 `json:"path"`
	Authentication string                 `json:"authentication"`
	Options        map[string]interface{} `json:"options"`
}

// NodeCredential represents authentication credentials for a node
type NodeCredential struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Webhook represents a webhook node from a workflow
type Webhook struct {
	// Core webhook information
	ID        string `json:"-"`
	WebhookID string `json:"id"`        // The webhook ID from n8n
	NodeID    string `json:"node_id"`   // ID of the node that contains this webhook
	NodeName  string `json:"node_name"` // Name of the node

	// Configuration
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	URL     string                 `json:"webhook_url"` // Full URL to access this webhook
	Options map[string]interface{} `json:"options"`
	Notes   string                 `json:"notes"` // notes, used to define automatically routes

	Route string `json:"route"` // route configuration for this webhook

	// Authentication
	AuthType    string              `json:"authentication"`
	Credentials *WebhookCredentials `json:"credentials,omitempty"`

	// Essential references - these are the minimum needed
	WorkflowID string `json:"workflow_id"`
	InstanceID string `json:"instance_id"`
}

// WebhookCredentials contains authentication details for a webhook
type WebhookCredentials struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InstanceStats represents statistics for an n8n instance
type InstanceStats struct {
	TotalWorkflows    int `json:"total"`
	ActiveWorkflows   int `json:"active"`
	InactiveWorkflows int `json:"inactive"`
	TotalWebhooks     int `json:"webhooks"`
	ActiveWebhooks    int `json:"active_webhooks"`
	InactiveWebhooks  int `json:"inactive_webhooks"`
	RedisTriggers     int `json:"redis"`
	ScheduledTriggers int `json:"scheduled"`
}

// API response types
type WorkflowsResponse struct {
	Data []Workflow `json:"data"`
}
