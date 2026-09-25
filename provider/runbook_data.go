package provider

import (
	"context"
	"fmt"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
)

// Singular services data source
func dataSourceRunbook() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataFireHydrantRunbook,
		Schema: map[string]*schema.Schema{
			// Required
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Computed
			"attachment_rule": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataFireHydrantRunbook(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the runbook
	runbookID := d.Get("id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Read runbook: %s", runbookID), map[string]interface{}{
		"id": runbookID,
	})
	response, err := client.Sdk.Runbooks.GetRunbook(ctx, runbookID)
	if err != nil {
		return diag.Errorf("Error reading runbook %s: %v", runbookID, err)
	}

	// Gather values from API response
	attributes := map[string]interface{}{
		"description": *response.Description,
		"name":        *response.Name,
	}

	if response.Owner != nil {
		attributes["owner_id"] = *response.Owner.ID
	}

	var attachmentRule []byte
	if response.AttachmentRule != nil {
		attachmentRule, err = response.AttachmentRule.MarshalJSON()
		if err != nil {
			return diag.Errorf("Error converting attachment_rule to JSON due invalid JSON returned by FireHydrant: %v", err)
		}
	}
	normalizedAttachmentRuleJSON, _ := structure.NormalizeJsonString(string(attachmentRule))
	attributes["attachment_rule"] = normalizedAttachmentRuleJSON

	// Set the data source attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for runbook %s: %v", key, runbookID, err)
		}
	}

	// Set the runbook's ID in state
	d.SetId(*response.ID)

	return diag.Diagnostics{}
}
