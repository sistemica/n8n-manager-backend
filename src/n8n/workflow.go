package n8n

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"go.uber.org/zap"
)

// syncWorkflows synchronizes workflows from an n8n instance to the PocketBase database
func syncWorkflows(app core.App, instance *Instance, workflows []Workflow, logger *zap.Logger) error {
	collection, err := app.FindCollectionByNameOrId("workflows")
	if err != nil {
		return fmt.Errorf("failed to find workflows collection: %w", err)
	}

	for _, workflow := range workflows {
		logger.Info("Processing workflow",
			zap.String("name", workflow.Name),
			zap.String("id", workflow.WorkflowID),
			zap.Bool("active", workflow.Active))

		// Get timestamps from the workflow to compare
		createdAt := workflow.CreatedAt.Format(time.RFC3339)
		updatedAt := workflow.UpdatedAt.Format(time.RFC3339)

		needsUpdate := true

		existingRecords, err := app.FindRecordsByFilter(
			collection,
			"workflow_id = {:workflow_id} && instance = {:instance}",
			"-created_at",
			1,
			0,
			dbx.Params{"instance": instance.Id},
			dbx.Params{"workflow_id": workflow.WorkflowID},
		)

		logger.Debug("Filter for database, found records?", zap.Int("records", len(existingRecords)))

		if err == nil && len(existingRecords) > 0 {
			logger.Debug("Found existing record for workflow, checking timestamps")
			existing := existingRecords[0]
			existingCreatedAt := existing.GetString("created_at")
			existingUpdatedAt := existing.GetString("updated_at")
			existingActive := existing.GetBool("active")

			logger.Debug("Workflow timestmaps",
				zap.String("workflow", workflow.WorkflowID),
				zap.String("existing_updated_at", existingUpdatedAt),
				zap.String("new_updated_at", updatedAt),
				zap.String("existing_created_at", existingCreatedAt),
				zap.String("new_created_at", createdAt),
				zap.Bool("existing_active", existingActive),
				zap.Bool("new_active", workflow.Active))

			// Skip if the workflow hasn't changed
			if existingCreatedAt == createdAt &&
				existingUpdatedAt == updatedAt &&
				existingActive == workflow.Active {
				logger.Debug("Workflow unchanged, skipping",
					zap.String("workflow", workflow.WorkflowID))
				needsUpdate = false
			} else {
				logger.Debug("Workflow changed, updating",
					zap.String("workflow", workflow.WorkflowID),
					zap.String("existing_updated_at", existingUpdatedAt),
					zap.String("new_updated_at", updatedAt),
					zap.Bool("existing_active", existingActive),
					zap.Bool("new_active", workflow.Active))
			}
		}

		if needsUpdate {
			// Create a new workflow record
			record := createWorkflowRecord(collection, instance, workflow)
			if err := app.Save(record); err != nil {
				logger.Error("Failed to save workflow",
					zap.String("workflow", workflow.WorkflowID),
					zap.Error(err))
				continue
			}

			// Sync webhooks to the database
			logger.Debug("Sync webhooks to database")
			if err := syncWebhooks(app, instance, workflow, logger); err != nil {
				logger.Error("Failed to sync webhooks",
					zap.Error(err),
					zap.String("instance", instance.Id))
			}

			logger.Info("Created new workflow version",
				zap.String("workflow", workflow.WorkflowID),
				zap.Bool("active", workflow.Active))
		}
	}

	return nil
}

// createWorkflowRecord creates a new record in the workflows collection
func createWorkflowRecord(
	collection *core.Collection,
	instance *Instance,
	workflow Workflow,
) *core.Record {
	record := core.NewRecord(collection)

	// Set parent instance reference
	record.Set("instance", instance.Id)

	// Set workflow metadata
	record.Set("workflow_name", workflow.Name)
	record.Set("workflow_id", workflow.WorkflowID)
	record.Set("created_at", workflow.CreatedAt.Format(time.RFC3339))
	record.Set("updated_at", workflow.UpdatedAt.Format(time.RFC3339))
	record.Set("number_of_nodes", len(workflow.Nodes))

	// Set node information
	record.Set("nodes", strings.Join(getNodeNames(workflow.Nodes), ","))

	// Store the full workflow data
	workflowData, _ := json.Marshal(workflow)
	record.Set("workflow_data", string(workflowData))
	record.Set("active", workflow.Active)

	t, _ := json.Marshal(workflow)
	fmt.Println(string(t))

	return record
}

// calculateWorkflowHash generates a hash of the workflow for comparison
func calculateWorkflowHash(workflow Workflow) (string, error) {
	data, err := json.Marshal(workflow)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// getNodeNames extracts the names of all nodes in a workflow
func getNodeNames(nodes []Node) []string {
	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name
	}
	return names
}
