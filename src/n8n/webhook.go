package n8n

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"go.uber.org/zap"
)

// syncWebhooks synchronizes webhooks from an n8n instance to the PocketBase database
func syncWebhooks(app core.App, instance *Instance, workflow Workflow, logger *zap.Logger) error {
	collection, err := app.FindCollectionByNameOrId("webhooks")
	if err != nil {
		return err
	}

	logger.Debug("Cleaning existing webhooks for instance",
		zap.String("host", instance.Host))

	// Delete existing webhooks for this instance
	records, err := app.FindAllRecords(collection, dbx.NewExp("instance = {:instance} AND workflow_id = {:workflow_id}",
		dbx.Params{
			"instance":    instance.Id,
			"workflow_id": workflow.WorkflowID,
		}))
	if err != nil {
		return err
	}

	logger.Debug("Found existing webhooks for instance", zap.Int("count", len(records)), zap.String("instance", instance.Id), zap.String("workflow", workflow.WorkflowID))
	for _, record := range records {
		if err = app.Delete(record); err != nil {
			return err
		}
	}

	webhooks := extractWebhooksFromWorkflow(workflow)

	// Insert new webhooks
	for _, webhook := range webhooks {
		// Get workflow name for logging (it's not in our model anymore)
		workflowName := getWorkflowName(app, instance.Id, workflow.WorkflowID)
		logger.Info("Creating webhook record",
			zap.String("workflow_id", workflow.WorkflowID),
			zap.String("workflow_name", workflowName),
			zap.String("node_id", webhook.NodeID))

		record := core.NewRecord(collection)

		// Set core fields
		record.Set("node_id", webhook.NodeID)
		record.Set("instance", instance.Id)
		record.Set("workflow_id", workflow.WorkflowID)
		record.Set("notes", webhook.Notes)
		record.Set("route", ExtractRoute(webhook.Notes))

		// We need to fetch the workflow name from the database
		// since it's not in our model anymore
		record.Set("workflow_name", workflowName)

		// Set webhook URL with the correct host
		record.Set("webhook_url", fmt.Sprintf("%s/webhook/%s", instance.Host, webhook.Path))

		// Process methods - always an array even if single method
		methods := []string{webhook.Method}
		methodsJson, err := json.Marshal(methods)
		if err != nil {
			logger.Error("Failed to marshal methods", zap.Error(err))
			continue
		}
		record.Set("methods", string(methodsJson))

		// Process options if available
		if webhook.Options != nil {
			optionsJson, err := json.Marshal(webhook.Options)
			if err != nil {
				logger.Error("Failed to marshal options", zap.Error(err))
				continue
			}
			record.Set("options", string(optionsJson))
		}

		// Set authentication if available
		if webhook.Credentials != nil {
			record.Set("auth_type", webhook.AuthType)
		}

		// Save the record
		if err := app.Save(record); err != nil {
			logger.Error("Failed to save webhook",
				zap.Error(err),
				zap.String("webhook_id", webhook.ID),
				zap.String("node_id", webhook.NodeID))
			continue
		}
	}

	return nil
}

// Helper function to get workflow name from workflow ID
func getWorkflowName(app core.App, instanceID, workflowID string) (name string) {
	records, err := app.FindRecordsByFilter(
		"workflows",
		"workflow_id = {:workflow_id} && instance = {:instance}",
		"-created_at",
		1,
		0,
		dbx.Params{"instance": instanceID},
		dbx.Params{"workflow_id": workflowID},
	)

	if err != nil || len(records) == 0 {
		return workflowID // Fall back to ID if workflow not found
	}

	return records[0].GetString("workflow_name")
}

// ExtractRoute finds the line containing "route:" and returns the trimmed part after it
func ExtractRoute(input string) string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "route:") {
			parts := strings.SplitN(line, "route:", 2)
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// Extract webhook information from a workflow
func extractWebhooksFromWorkflow(workflow Workflow) []Webhook {
	var webhooks []Webhook
	for _, node := range workflow.Nodes {
		if node.Type != "n8n-nodes-base.webhook" {
			continue
		}

		// Create webhook with only the fields available in our model
		webhook := Webhook{
			ID:       node.WebhookID,
			NodeID:   node.ID,
			NodeName: node.Name,
			Method:   node.Parameters.HTTPMethod,
			Path:     node.Parameters.Path,
			AuthType: node.Parameters.Authentication,
			Options:  node.Parameters.Options,
			Notes:    node.Notes,
			// Set essential references
			WorkflowID: workflow.ID,
			InstanceID: workflow.InstanceID,
			// Build URL
			URL: fmt.Sprintf("%s/webhook/%s", workflow.InstanceID, node.Parameters.Path),
		}

		if node.Credentials != nil {
			for credType, cred := range node.Credentials {
				webhook.Credentials = &WebhookCredentials{
					Type: credType,
					ID:   cred.ID,
					Name: cred.Name,
				}
				break // Only use the first credential
			}
		}

		webhooks = append(webhooks, webhook)
	}
	return webhooks
}
