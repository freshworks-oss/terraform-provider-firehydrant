package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSeverity() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceFireHydrantSeverity,
		UpdateContext: updateResourceFireHydrantSeverity,
		ReadContext:   readResourceFireHydrantSeverity,
		DeleteContext: deleteResourceFireHydrantSeverity,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// Required
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k string, oldValue string, newValue string, d *schema.ResourceData) bool {
					// Slug is case-insensitive, so don't show a diff if the string are the same when compared
					// in all lowercase
					if strings.EqualFold(oldValue, newValue) {
						return true
					}
					return false
				},
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.All(
						validation.StringLenBetween(0, 23),
						validation.StringMatch(regexp.MustCompile(`\A[[:alnum:]-]+\z`), "must only include letters, numbers, and hyphens"),
					),
				),
			},

			// Optional
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  string(firehydrant.SeverityTypeUnexpectedDowntime),
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringInSlice(
						[]string{
							string(firehydrant.SeverityTypeGameday),
							string(firehydrant.SeverityTypeMaintenance),
							string(firehydrant.SeverityTypeUnexpectedDowntime),
						},
						false,
					),
				),
			},
		},
	}
}

func readResourceFireHydrantSeverity(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	firehydrantAPIClient := m.(firehydrant.Client)

	// Get the severity
	severityID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read severity: %s", severityID), map[string]interface{}{
		"id": severityID,
	})
	severityResponse, err := firehydrantAPIClient.Severities().Get(ctx, severityID)
	if err != nil {
		if errors.Is(err, firehydrant.ErrorNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Severity %s no longer exists", severityID), map[string]interface{}{
				"id": severityID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading severity %s: %v", severityID, err)
	}

	// Gather values from API response
	attributes := map[string]interface{}{
		"slug":        severityResponse.Slug,
		"description": severityResponse.Description,
		"type":        severityResponse.Type,
	}

	// Set the resource attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for severity %s: %v", key, severityID, err)
		}
	}

	return diag.Diagnostics{}
}

func createResourceFireHydrantSeverity(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	firehydrantAPIClient := m.(firehydrant.Client)

	// Get attributes from config and construct the create request
	createRequest := firehydrant.CreateSeverityRequest{
		Slug:        d.Get("slug").(string),
		Description: d.Get("description").(string),
		Type:        d.Get("type").(string),
	}

	// Create the new severity
	tflog.Debug(ctx, fmt.Sprintf("Create severity: %s", createRequest.Slug), map[string]interface{}{
		"slug": createRequest.Slug,
	})
	severityResponse, err := firehydrantAPIClient.Severities().Create(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating severity %s: %v", createRequest.Slug, err)
	}

	// Set the new severity's ID in state
	d.SetId(severityResponse.Slug)

	// Update state with the latest information from the API
	return readResourceFireHydrantSeverity(ctx, d, m)
}

func updateResourceFireHydrantSeverity(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	firehydrantAPIClient := m.(firehydrant.Client)

	// Construct the update request
	updateRequest := firehydrant.UpdateSeverityRequest{
		Slug:        d.Get("slug").(string),
		Description: d.Get("description").(string),
		Type:        d.Get("type").(string),
	}

	// Update the severity
	tflog.Debug(ctx, fmt.Sprintf("Update severity: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	_, err := firehydrantAPIClient.Severities().Update(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.Errorf("Error updating severity %s: %v", d.Id(), err)
	}

	// Update state with the latest information from the API
	return readResourceFireHydrantSeverity(ctx, d, m)
}

func deleteResourceFireHydrantSeverity(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	firehydrantAPIClient := m.(firehydrant.Client)

	// Delete the severity
	severityID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Delete severity: %s", severityID), map[string]interface{}{
		"id": severityID,
	})
	err := firehydrantAPIClient.Severities().Delete(ctx, severityID)
	if err != nil {
		if errors.Is(err, firehydrant.ErrorNotFound) {
			return nil
		}
		return diag.Errorf("Error deleting severity %s: %v", severityID, err)
	}

	return diag.Diagnostics{}
}
