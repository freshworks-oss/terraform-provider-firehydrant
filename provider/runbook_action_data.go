package provider

import (
	"context"
	"fmt"

	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRunbookAction() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataFireHydrantRunbookAction,
		Schema: map[string]*schema.Schema{
			// Required
			"integration_slug": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Optional
			"type": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Computed
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataFireHydrantRunbookAction(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the runbook action
	runbookType := d.Get("type").(string)
	actionSlug := d.Get("slug").(string)
	integrationSlug := d.Get("integration_slug").(string)
	tflog.Debug(ctx, fmt.Sprintf("Read runbook action: %s:%s", integrationSlug, actionSlug), map[string]interface{}{
		"type":             runbookType,
		"slug":             actionSlug,
		"integration_slug": integrationSlug,
	})

	// these values were hardcoded into the REST client, so we'll continue to use them here
	perPage := 100
	isLite := true
	listResponse, err := client.Sdk.Runbooks.ListRunbookActions(ctx, nil, &perPage, &runbookType, &isLite)
	if err != nil {
		return diag.Errorf("Error getting runbook actions list: %v", err)
	}

	var requestedAction *components.RunbooksActionsEntity
	for _, action := range listResponse.Data {
		if *action.Slug == actionSlug && *action.Integration.Slug == integrationSlug {
			requestedAction = &action
		}
	}

	if requestedAction == nil {
		return diag.Errorf("Error reading runbook action %s:%s: %v", integrationSlug, actionSlug, err)
	}

	// Update the attributes in state to the values we got from the API
	attributes := map[string]string{
		"name": *requestedAction.Name,
		"slug": *requestedAction.Slug,
	}

	if requestedAction.Integration != nil {
		attributes["integration_slug"] = *requestedAction.Integration.Slug
	}

	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for runbook action %s:%s: %v", key, integrationSlug, actionSlug, err)
		}
	}

	// Set the runbook action's ID in state
	d.SetId(*requestedAction.ID)

	return diag.Diagnostics{}
}
