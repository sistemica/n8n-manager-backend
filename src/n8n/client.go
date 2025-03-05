package n8n

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const API_PATH = "/api/v1/"

// Client represents an HTTP client with configuration
type Client struct {
	http    *http.Client
	timeout time.Duration
}

// NewClient creates a new HTTP client with default configuration
func NewClient() *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// API paths
func (instance *Instance) GetApiPath() string {
	return instance.Host + API_PATH
}

func (instance *Instance) GetWorkflowsPath() string {
	return instance.GetApiPath() + "workflows"
}

func (instance *Instance) GetStatusPath() string {
	return instance.GetApiPath() + "health"
}

func (instance *Instance) GetWebhooksPath() string {
	return instance.GetApiPath() + "workflows/webhook"
}

func (instance *Instance) GetActivationPath(workflowId string) string {
	return fmt.Sprintf("%sworkflows/%s/activate", instance.GetApiPath(), workflowId)
}

func (instance *Instance) GetDeactivationPath(workflowId string) string {
	return fmt.Sprintf("%sworkflows/%s/deactivate", instance.GetApiPath(), workflowId)
}

// newRequest creates a new HTTP request with common headers
func (instance *Instance) newRequest(method, path string) (*http.Request, error) {
	url := instance.GetApiPath() + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add("X-N8N-API-KEY", instance.APIKey)
	return req, nil
}

// IsHealthy checks if the n8n instance is healthy
func (instance *Instance) IsHealthy() bool {
	req, err := instance.newRequest("GET", "health")
	if err != nil {
		return false
	}

	client := NewClient()
	resp, err := client.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetWorkflows retrieves all workflows from the n8n instance
func (instance *Instance) GetWorkflows() ([]Workflow, error) {
	req, err := instance.newRequest("GET", "workflows")
	if err != nil {
		return nil, err
	}

	client := NewClient()
	resp, err := client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Optional: Save response to file for debugging
	if debugMode := os.Getenv("N8N_DEBUG"); debugMode == "true" {
		debugFile := CreateFileNameFromHost(instance.Host) + "_workflows.json"
		err := os.WriteFile(debugFile, responseBytes, 0644)
		if err != nil {
			return nil, fmt.Errorf("error writing debug file: %w", err)
		}
	}

	var response WorkflowsResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Add instance ID to each workflow
	for idx := range response.Data {
		response.Data[idx].InstanceID = instance.Id
	}

	return response.Data, nil
}

// GetWorkflow retrieves a specific workflow by ID
func (instance *Instance) GetWorkflow(id string) (*Workflow, error) {
	req, err := instance.newRequest("GET", fmt.Sprintf("workflows/%s", id))
	if err != nil {
		return nil, err
	}

	client := NewClient()
	resp, err := client.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var workflow Workflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Set instance ID
	workflow.InstanceID = instance.Id

	return &workflow, nil
}

// DownloadWorkflows downloads all workflows and returns them as a map of filename to JSON content
func (instance *Instance) DownloadWorkflows() (map[string][]byte, error) {
	workflows, err := instance.GetWorkflows()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, workflow := range workflows {
		cleanName := cleanString(workflow.Name)
		filename := fmt.Sprintf("workflow_%s.json", cleanName)

		fullWorkflow, err := instance.GetWorkflow(workflow.ID)
		if err != nil {
			return nil, fmt.Errorf("error downloading workflow %s: %w", workflow.ID, err)
		}

		jsonData, err := json.MarshalIndent(fullWorkflow, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling workflow %s: %w", workflow.ID, err)
		}

		result[filename] = jsonData
	}

	return result, nil
}

// Utility functions
func cleanString(s string) string {
	clean := make([]byte, len(s))
	j := 0
	for _, b := range []byte(s) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			clean[j] = b
			j++
		} else if b == ' ' {
			clean[j] = '_'
			j++
		}
	}
	return string(clean[:j])
}

func CreateFileNameFromHost(s string) string {
	return cleanString(strings.ReplaceAll(strings.ReplaceAll(s, ".", "_"), "https://", ""))
}
