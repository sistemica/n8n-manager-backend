package migrations

import (
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Configure application settings
		settings := app.Settings()

		// for all available settings fields you could check
		// https://github.com/pocketbase/pocketbase/blob/develop/core/settings_model.go#L121-L130
		settings.Meta.AppName = "n8n Monitor"
		settings.Meta.AppURL = "https://example.com"
		settings.Logs.MaxDays = 2
		settings.Logs.LogAuthId = true
		settings.Logs.LogIP = false

		if err := app.Save(settings); err != nil {
			return err
		}

		// Create admin superuser
		superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		adminRecord := core.NewRecord(superusers)
		adminRecord.Set("email", "change@me.com")
		adminRecord.Set("password", "change@me.com")

		if err := app.Save(adminRecord); err != nil {
			return err
		}

		// Create regular user
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		userRecord := core.NewRecord(users)
		userRecord.Set("email", "user@example.com")
		userRecord.Set("password", "user@example.com")
		userRecord.Set("name", "Regular User")
		userRecord.Set("verified", true)

		if err := app.Save(userRecord); err != nil {
			return err
		}

		// Create the instance collection - parent in our hierarchy
		instancesCollection := core.NewBaseCollection("instances")
		instancesCollection.Fields.Add(
			&core.TextField{
				Name:     "host",
				Required: true,
			},
			&core.BoolField{
				Name: "ignore_ssl_errors",
			},
			&core.TextField{
				Name:     "api_key",
				Required: true,
			},
			&core.NumberField{
				Name: "check_interval_mins",
			},
			&core.DateField{
				Name: "last_check",
			},
			&core.BoolField{
				Name: "availability_status",
			},
			&core.TextField{
				Name:     "availability_note",
				Required: false,
			},
			&core.NumberField{
				Name: "workflows_active",
			},
			&core.NumberField{
				Name: "workflows_inactive",
			},
			&core.NumberField{
				Name: "webhooks_active",
			},
			&core.NumberField{
				Name: "webhooks_inactive",
			},
		)

		if err := app.Save(instancesCollection); err != nil {
			return err
		}

		// Create the workflow collection - child of instance
		workflowsCollection := core.NewBaseCollection("workflows")
		workflowsCollection.Fields.Add(
			&core.RelationField{
				Name:          "instance",
				Required:      true,
				CascadeDelete: true,
				CollectionId:  instancesCollection.Id,
				MaxSelect:     1,
			},
			&core.TextField{
				Name:     "workflow_name",
				Required: true,
			},
			&core.TextField{
				Name:     "workflow_id",
				Required: true,
			},
			&core.TextField{
				Name:     "created_at",
				Required: true,
			},
			&core.TextField{
				Name:     "updated_at",
				Required: true,
			},
			&core.NumberField{
				Name:     "number_of_nodes",
				Required: true,
			},
			&core.JSONField{
				Name: "workflow_data",
			},
			&core.TextField{
				Name: "nodes",
			},
			&core.BoolField{
				Name:     "active",
				Required: false,
			},
		)

		if err := app.Save(workflowsCollection); err != nil {
			return err
		}

		// Create the webhooks collection - child of workflow and instance
		webhooksCollection := core.NewBaseCollection("webhooks")
		webhooksCollection.Fields.Add(
			&core.RelationField{
				Name:          "instance",
				Required:      true,
				CascadeDelete: true,
				CollectionId:  instancesCollection.Id,
				MaxSelect:     1,
			},
			&core.TextField{
				Name:     "workflow_name",
				Required: true,
			},
			&core.TextField{
				Name:     "workflow_id",
				Required: true,
			},
			&core.TextField{
				Name:     "node_id",
				Required: true,
			},
			&core.TextField{
				Name:     "webhook_url",
				Required: true,
			},
			&core.JSONField{
				Name:     "methods",
				Required: true,
			},
			&core.JSONField{
				Name: "options",
			},
			&core.JSONField{
				Name: "parameters",
			},
			&core.TextField{
				Name:     "auth_type",
				Required: false,
			},
			&core.TextField{
				Name:     "route",
				Required: false,
			},
			&core.TextField{
				Name:     "notes",
				Required: false,
			},
		)

		if err := app.Save(webhooksCollection); err != nil {
			return err
		}

		// Create a sample instance record
		instanceRecord := core.NewRecord(instancesCollection)
		instanceRecord.Set("host", "https://n8n.integrate-now.com")
		instanceRecord.Set("api_key", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIzMTgwMWNhNS05ZDQ2LTQyZTUtYWZlZS02ZjQzOWNlMzIyNmIiLCJpc3MiOiJuOG4iLCJhdWQiOiJwdWJsaWMtYXBpIiwiaWF0IjoxNzM4ODM1OTEyfQ.PzdqqLR7p1t_8f--CIiRcPU8zPec4rlnyHow-lPhCBU")
		instanceRecord.Set("check_interval_mins", 1) // Check every minute
		instanceRecord.Set("ignore_ssl_errors", false)

		if err := app.Save(instanceRecord); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollback function
		fmt.Println("Rolling back migration")

		// In PocketBase, we need to handle collection deletion manually
		// Currently, this is a placeholder since proper rollback would require
		// iterating through records and deleting them

		fmt.Println("Proper rollback not implemented yet")
		return errors.New("Migration rollback not implemented")
	})
}
