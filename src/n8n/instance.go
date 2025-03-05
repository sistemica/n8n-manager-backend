package n8n

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"go.uber.org/zap"
)

// GetInstanceStats collects statistics about an n8n instance
func (instance *Instance) GetInstanceStats(workflows []Workflow) (*InstanceStats, error) {
	stats := &InstanceStats{}
	stats.TotalWorkflows = len(workflows)

	// Extract webhooks and count different types of nodes
	for _, workflow := range workflows {
		if workflow.Active {
			stats.ActiveWorkflows++
		} else {
			stats.InactiveWorkflows++
		}

		for _, node := range workflow.Nodes {
			switch node.Type {
			case "n8n-nodes-base.webhook":
				stats.TotalWebhooks++
				if workflow.Active {
					stats.ActiveWebhooks++
				} else {
					stats.InactiveWebhooks++
				}
			case "n8n-nodes-base.redisTrigger":
				stats.RedisTriggers++
			case "n8n-nodes-base.scheduleTrigger":
				stats.ScheduledTriggers++
			}
		}

	}

	return stats, nil
}

// shouldCheckInstance determines if it's time to check an instance based on its check interval
func shouldCheckInstance(lastCheck types.DateTime, checkInterval int) bool {
	if checkInterval == 0 {
		checkInterval = 5 // Default to 5 minutes
	}

	lastCheckTime := lastCheck.Time()
	nextCheckTime := lastCheckTime.Add(time.Duration(checkInterval) * time.Minute)
	return time.Now().After(nextCheckTime)
}

// InitCronJobs sets up the recurring check of n8n instances
func InitCronJobs(app core.App, logger *zap.Logger) {
	app.Cron().MustAdd("check-instances", "* * * * *", func() {
		instances, err := app.FindAllRecords("instances")
		if err != nil {
			logger.Error("Failed to fetch n8n instances", zap.Error(err))
			return
		}

		for _, record := range instances {
			// Skip if it's not time to check this instance yet
			if !shouldCheckInstance(record.GetDateTime("last_check"), record.GetInt("check_interval_mins")) {
				continue
			}

			// Create instance object
			instance := NewInstance(
				record.Id,
				record.GetString("host"),
				record.GetString("api_key"),
			)

			// Set instance's check interval from DB
			instance.CheckInterval = record.GetInt("check_interval_mins")
			instance.IgnoreSSLErrors = record.GetBool("ignore_ssl_errors")

			// Start the sync process
			err := syncInstance(app, instance, record, logger)
			if err != nil {
				logger.Error("Failed to sync instance",
					zap.Error(err),
					zap.String("instance", instance.Host))

				// Update record with error information
				record.Set("last_check", time.Now())
				record.Set("availability_status", false)
				record.Set("availability_note", err.Error())
				if saveErr := app.Save(record); saveErr != nil {
					logger.Error("Failed to update instance status", zap.Error(saveErr))
				}
			}
		}
	})
}

// syncInstance handles the complete sync process for a single instance
func syncInstance(app core.App, instance *Instance, record *core.Record, logger *zap.Logger) error {
	// Fetch all workflows from the instance
	workflows, err := instance.GetWorkflows()
	if err != nil {
		return fmt.Errorf("failed to get workflows: %w", err)
	}

	// Get statistics based on the workflows
	stats, err := instance.GetInstanceStats(workflows)
	if err != nil {
		return fmt.Errorf("failed to calculate instance statistics: %w", err)
	}

	// Sync workflows to the database
	if err := syncWorkflows(app, instance, workflows, logger); err != nil {
		logger.Error("Failed to sync workflows",
			zap.Error(err),
			zap.String("instance", instance.Id))
	}

	// Update instance record with new statistics
	record.Set("workflows_active", stats.ActiveWorkflows)
	record.Set("workflows_inactive", stats.InactiveWorkflows)
	record.Set("webhooks_active", stats.ActiveWebhooks)
	record.Set("webhooks_inactive", stats.InactiveWebhooks)
	record.Set("last_check", time.Now())
	record.Set("availability_status", true)
	record.Set("availability_note", "")

	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to update instance record: %w", err)
	}

	logger.Info("Updated n8n instance",
		zap.String("instance", record.Id),
		zap.Int("total_workflows", stats.TotalWorkflows),
		zap.Int("active_workflows", stats.ActiveWorkflows),
		zap.Int("webhooks", stats.TotalWebhooks))

	return nil
}
