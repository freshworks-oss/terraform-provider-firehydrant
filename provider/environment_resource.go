package provider

import (
	"context"
	"fmt"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/firehydrant-go-sdk/models/sdkerrors"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceEnvironment() *schema.Resource {
	return &schema.Resource{
		Description:   "FireHydrant environments are used to tag incidents with where they are occurring.",
		CreateContext: createResourceFireHydrantEnvironment,
		UpdateContext: updateResourceFireHydrantEnvironment,
		ReadContext:   readResourceFireHydrantEnvironment,
		DeleteContext: deleteResourceFireHydrantEnvironment,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Optional
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func readResourceFireHydrantEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the environment
	environmentID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read environment: %s", environmentID), map[string]interface{}{
		"id": environmentID,
	})
	environmentResponse, err := client.Sdk.CatalogEntries.GetEnvironment(ctx, environmentID)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			tflog.Debug(ctx, fmt.Sprintf("Environment %s no longer exists", environmentID), map[string]interface{}{
				"id": environmentID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading environment %s: %v", environmentID, err)
	}

	// Gather values from API response
	attributes := map[string]interface{}{
		"name":        *environmentResponse.Name,
		"description": *environmentResponse.Description,
	}

	// Set the resource attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for environment %s: %v", key, environmentID, err)
		}
	}

	return diag.Diagnostics{}
}

func createResourceFireHydrantEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get attributes from config and construct the create request
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	createRequest := components.CreateEnvironment{
		Name:        name,
		Description: &description,
	}

	// Create the new environment
	tflog.Debug(ctx, fmt.Sprintf("Create environment: %s", name), map[string]interface{}{
		"name": name,
	})
	environmentResponse, err := client.Sdk.CatalogEntries.CreateEnvironment(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating environment %s: %v", name, err)
	}

	// Set the new environment's ID in state
	d.SetId(*environmentResponse.ID)

	// Update state with the latest information from the API
	return readResourceFireHydrantEnvironment(ctx, d, m)
}

func updateResourceFireHydrantEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Construct the update request
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	updateRequest := components.UpdateEnvironment{
		Name:        &name,
		Description: &description,
	}

	// Update the environment
	tflog.Debug(ctx, fmt.Sprintf("Update environment: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	_, err := client.Sdk.CatalogEntries.UpdateEnvironment(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.Errorf("Error updating environment %s: %v", d.Id(), err)
	}

	// Update state with the latest information from the API
	return readResourceFireHydrantEnvironment(ctx, d, m)
}

func deleteResourceFireHydrantEnvironment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Delete the environment
	environmentID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Delete environment: %s", environmentID), map[string]interface{}{
		"id": environmentID,
	})
	err := client.Sdk.CatalogEntries.DeleteEnvironment(ctx, environmentID)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			return nil
		}
		return diag.Errorf("Error deleting environment %s: %v", environmentID, err)
	}

	return diag.Diagnostics{}
}
