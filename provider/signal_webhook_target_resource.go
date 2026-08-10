package provider

import (
	"context"
	"fmt"

	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/firehydrant-go-sdk/models/sdkerrors"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSignalWebhookTarget() *schema.Resource {
	return &schema.Resource{
		Description:   "FireHydrant signal webhook targets are URLs that FireHydrant notifies when signals are received.",
		CreateContext: createResourceFireHydrantSignalWebhookTarget,
		UpdateContext: updateResourceFireHydrantSignalWebhookTarget,
		ReadContext:   readResourceFireHydrantSignalWebhookTarget,
		DeleteContext: deleteResourceFireHydrantSignalWebhookTarget,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the webhook target.",
			},
			"url": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The URL that the webhook target will notify.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithHTTPorHTTPS),
			},

			// Optional
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A description of the webhook target.",
			},
			"signing_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				Description: "A secret FireHydrant provides in the FH-Signature header when sending payloads to the " +
					"webhook target. The API never returns this value once it has been set, so it is only ever sent to " +
					"FireHydrant, never read back.",
			},
		},
	}
}

func readResourceFireHydrantSignalWebhookTarget(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the signal webhook target
	webhookTargetID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read signal webhook target: %s", webhookTargetID), map[string]interface{}{
		"id": webhookTargetID,
	})
	webhookTargetResponse, err := client.Sdk.Signals.GetSignalsWebhookTarget(ctx, webhookTargetID)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			tflog.Debug(ctx, fmt.Sprintf("Signal webhook target %s no longer exists", webhookTargetID), map[string]interface{}{
				"id": webhookTargetID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading signal webhook target %s: %v", webhookTargetID, err)
	}

	// Process any data that could be nil. The SDK models every field as a
	// pointer, so an absent value becomes an empty string rather than a panic.
	// Absent values are still written to state so that a change made outside of
	// Terraform shows up as a diff instead of lingering in state.
	var name, url, description string
	if webhookTargetResponse.Name != nil {
		name = *webhookTargetResponse.Name
	}
	if webhookTargetResponse.URL != nil {
		url = *webhookTargetResponse.URL
	}
	if webhookTargetResponse.Description != nil {
		description = *webhookTargetResponse.Description
	}

	// Gather values from API response.
	//
	// signing_key is deliberately absent: the API does not return it once it has
	// been set, so reading it back would wipe the configured value out of state
	// on every refresh.
	attributes := map[string]interface{}{
		"name":        name,
		"url":         url,
		"description": description,
	}

	// Set the resource attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for signal webhook target %s: %v", key, webhookTargetID, err)
		}
	}

	return diag.Diagnostics{}
}

func createResourceFireHydrantSignalWebhookTarget(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get attributes from config and construct the create request
	createRequest := components.CreateSignalsWebhookTarget{
		Name: d.Get("name").(string),
		URL:  d.Get("url").(string),
	}

	// Process any optional attributes and add to the create request if necessary
	if description, ok := d.GetOk("description"); ok {
		value := description.(string)
		createRequest.Description = &value
	}
	if signingKey, ok := d.GetOk("signing_key"); ok {
		value := signingKey.(string)
		createRequest.SigningKey = &value
	}

	// Create the new signal webhook target
	tflog.Debug(ctx, fmt.Sprintf("Create signal webhook target: %s", createRequest.Name), map[string]interface{}{
		"name": createRequest.Name,
	})
	webhookTargetResponse, err := client.Sdk.Signals.CreateSignalsWebhookTarget(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating signal webhook target %s: %v", createRequest.Name, err)
	}
	if webhookTargetResponse.ID == nil {
		return diag.Errorf("Error creating signal webhook target %s: the API response did not include an ID", createRequest.Name)
	}

	// Set the new signal webhook target's ID in state
	d.SetId(*webhookTargetResponse.ID)

	// Update state with the latest information from the API
	return readResourceFireHydrantSignalWebhookTarget(ctx, d, m)
}

func updateResourceFireHydrantSignalWebhookTarget(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Construct the update request. name, url, and description are always sent,
	// including when description is empty, so that removing the description from
	// the configuration also removes it in FireHydrant.
	name := d.Get("name").(string)
	url := d.Get("url").(string)
	description := d.Get("description").(string)
	updateRequest := components.UpdateSignalsWebhookTarget{
		Name:        &name,
		URL:         &url,
		Description: &description,
	}

	// Only send the signing key when it has changed, so that unrelated updates
	// do not send the secret again. Removing signing_key from the configuration
	// leaves the key that is already configured in FireHydrant in place.
	if d.HasChange("signing_key") {
		if signingKey := d.Get("signing_key").(string); signingKey != "" {
			updateRequest.SigningKey = &signingKey
		}
	}

	// Update the signal webhook target
	webhookTargetID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Update signal webhook target: %s", webhookTargetID), map[string]interface{}{
		"id": webhookTargetID,
	})
	_, err := client.Sdk.Signals.UpdateSignalsWebhookTarget(ctx, webhookTargetID, updateRequest)
	if err != nil {
		return diag.Errorf("Error updating signal webhook target %s: %v", webhookTargetID, err)
	}

	// Update state with the latest information from the API
	return readResourceFireHydrantSignalWebhookTarget(ctx, d, m)
}

func deleteResourceFireHydrantSignalWebhookTarget(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Delete the signal webhook target
	webhookTargetID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Delete signal webhook target: %s", webhookTargetID), map[string]interface{}{
		"id": webhookTargetID,
	})
	err := client.Sdk.Signals.DeleteSignalsWebhookTarget(ctx, webhookTargetID)
	if err != nil {
		if !signalsDeleteErrorMeansGone(err) {
			return diag.Errorf("Error deleting signal webhook target %s: %v", webhookTargetID, err)
		}
		tflog.Debug(ctx, fmt.Sprintf("Signal webhook target %s is gone: %v", webhookTargetID, err), map[string]interface{}{
			"id": webhookTargetID,
		})
	}

	d.SetId("")

	return diag.Diagnostics{}
}

// signalsDeleteErrorMeansGone reports whether an error from a Signals delete
// endpoint actually means the resource is gone.
//
// The delete operations in the SDK treat 204 as the only success and report every
// other status, success included, as an unknown status code. Signals delete
// endpoints answer 200 with the resource they deleted, so a delete that worked
// comes back as an error carrying a 2xx status. Deleting something that is already
// gone answers 404, which only needs to come out of state.
func signalsDeleteErrorMeansGone(err error) bool {
	sdkErr, ok := err.(*sdkerrors.SDKError)
	if !ok {
		return false
	}

	if sdkErr.StatusCode >= 200 && sdkErr.StatusCode < 300 {
		return true
	}

	return sdkErr.StatusCode == 404
}
