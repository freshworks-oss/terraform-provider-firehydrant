package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceSignalRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceFireHydrantSignalRule,
		UpdateContext: updateResourceFireHydrantSignalRule,
		ReadContext:   readResourceFireHydrantSignalRule,
		DeleteContext: deleteResourceFireHydrantSignalRule,
		Importer: &schema.ResourceImporter{
			StateContext: importResourceFireHydrantSignalRule,
		},
		Schema: map[string]*schema.Schema{
			// Required
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"expression": {
				Type:     schema.TypeString,
				Required: true,
			},
			"target_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"target_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"incident_type_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"notification_priority_override": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(firehydrant.NotificationPriorityLow),
					string(firehydrant.NotificationPriorityMedium),
					string(firehydrant.NotificationPriorityHigh),
				}, false),
			},
			"create_incident_condition_when": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(firehydrant.CreateIncidentConditionWhenUnspecified),
					string(firehydrant.CreateIncidentConditionWhenAlways),
				}, false),
			},
			"deduplication_expiry": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Duration for deduplicating similar alerts (ISO8601 duration format e.g., 'PT30M', 'PT2H', 'P1D')",
			},
			// Target fields for additional information
			"target_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_team_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target_is_pageable": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func readResourceFireHydrantSignalRule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the signal rule
	id := d.Id()
	teamID := d.Get("team_id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Read signal rule: %s", id), map[string]interface{}{
		"id":      id,
		"team_id": teamID,
	})

	signalRule, err := client.Sdk.Signals.GetTeamSignalRule(ctx, teamID, id)
	if err != nil {
		if errors.Is(err, firehydrant.ErrorNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("Signal rule %s no longer exists", id), map[string]interface{}{
				"id":      id,
				"team_id": teamID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading signal rule %s: %v", id, err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Read signal rule %s", id), map[string]interface{}{
		"id":                             id,
		"notification_priority_override": signalRule.GetNotificationPriorityOverride(),
	})

	// Gather values from API response
	target := signalRule.GetTarget()
	attributes := map[string]interface{}{
		"name":        stringValue(signalRule.GetName()),
		"expression":  stringValue(signalRule.GetExpression()),
		"target_type": stringValue(target.GetType()),
		"target_id":   stringValue(target.GetID()),
	}

	// Handle target additional fields
	if target != nil {
		if target.GetName() != nil {
			attributes["target_name"] = *target.GetName()
		}
		// Only set target_team_id for certain target types (escalation policies, teams)
		if target.GetTeamID() != nil && target.GetType() != nil {
			targetType := *target.GetType()
			if targetType == "escalation_policy" || targetType == "team" {
				attributes["target_team_id"] = *target.GetTeamID()
			}
		}
		if target.GetIsPageable() != nil {
			attributes["target_is_pageable"] = *target.GetIsPageable()
		}
	}

	// Handle incident type
	if incidentType := signalRule.GetIncidentType(); incidentType != nil && incidentType.GetID() != nil {
		attributes["incident_type_id"] = *incidentType.GetID()
	}

	// Handle notification priority override
	if priority := signalRule.GetNotificationPriorityOverride(); priority != nil && string(*priority) != "" {
		attributes["notification_priority_override"] = string(*priority)
	}

	// Handle create incident condition when
	if condition := signalRule.GetCreateIncidentConditionWhen(); condition != nil {
		attributes["create_incident_condition_when"] = string(*condition)
	}

	// Handle deduplication expiry
	if expiry := signalRule.GetDeduplicationExpiry(); expiry != nil {
		attributes["deduplication_expiry"] = *expiry
	}

	// Set the resource attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for signal rule %s: %v", key, id, err)
		}
	}

	return diag.Diagnostics{}
}

// importResourceFireHydrantSignalRule imports a signal rule given an ID of the form
// Team_ID:Signal_Rule_ID, since signal rules are scoped to a team and a bare rule ID
// is not enough to look one up.
func importResourceFireHydrantSignalRule(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	teamID, id, err := resourceFireHydrantSignalRuleParseId(d.Id())
	if err != nil {
		return nil, err
	}

	d.Set("team_id", teamID)
	d.SetId(id)

	return []*schema.ResourceData{d}, nil
}

func resourceFireHydrantSignalRuleParseId(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected Team_ID:Signal_Rule_ID", id)
	}

	return parts[0], parts[1], nil
}

func createResourceFireHydrantSignalRule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Construct the create request
	name := d.Get("name").(string)
	expression := d.Get("expression").(string)
	targetType := d.Get("target_type").(string)
	targetID := d.Get("target_id").(string)

	createRequest := components.CreateTeamSignalRule{
		Name:       name,
		Expression: expression,
		TargetType: components.CreateTeamSignalRuleTargetType(targetType),
		TargetID:   targetID,
	}

	// Handle optional fields
	if incidentTypeID := d.Get("incident_type_id").(string); incidentTypeID != "" {
		createRequest.IncidentTypeID = &incidentTypeID
	}

	if priority := d.Get("notification_priority_override").(string); priority != "" {
		priorityEnum := components.CreateTeamSignalRuleNotificationPriorityOverride(priority)
		createRequest.NotificationPriorityOverride = &priorityEnum
	}

	if condition := d.Get("create_incident_condition_when").(string); condition != "" {
		conditionEnum := components.CreateTeamSignalRuleCreateIncidentConditionWhen(condition)
		createRequest.CreateIncidentConditionWhen = &conditionEnum
	}

	if expiry := d.Get("deduplication_expiry").(string); expiry != "" {
		createRequest.DeduplicationExpiry = &expiry
	}

	// Create the signal rule
	tflog.Debug(ctx, fmt.Sprintf("Create signal rule: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	signalRuleResponse, err := client.Sdk.Signals.CreateTeamSignalRule(ctx, d.Get("team_id").(string), createRequest)
	if err != nil {
		return diag.Errorf("Error creating signal rule %s: %v", d.Id(), err)
	}

	// Set the ID of the resource to the ID of the newly created signal rule
	d.SetId(*signalRuleResponse.GetID())

	// Update state with the latest information from the API
	return readResourceFireHydrantSignalRule(ctx, d, m)
}

func updateResourceFireHydrantSignalRule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Construct the update request
	name := d.Get("name").(string)
	expression := d.Get("expression").(string)
	targetType := d.Get("target_type").(string)
	targetID := d.Get("target_id").(string)

	updateRequest := components.UpdateTeamSignalRule{
		Name:       &name,
		Expression: &expression,
		TargetType: (*components.UpdateTeamSignalRuleTargetType)(&targetType),
		TargetID:   &targetID,
	}

	// Handle optional fields
	if incidentTypeID := d.Get("incident_type_id").(string); incidentTypeID != "" {
		updateRequest.IncidentTypeID = &incidentTypeID
	}

	if priority := d.Get("notification_priority_override").(string); priority != "" {
		priorityEnum := components.UpdateTeamSignalRuleNotificationPriorityOverride(priority)
		updateRequest.NotificationPriorityOverride = &priorityEnum
	}

	if condition := d.Get("create_incident_condition_when").(string); condition != "" {
		conditionEnum := components.UpdateTeamSignalRuleCreateIncidentConditionWhen(condition)
		updateRequest.CreateIncidentConditionWhen = &conditionEnum
	}

	if expiry := d.Get("deduplication_expiry").(string); expiry != "" {
		updateRequest.DeduplicationExpiry = &expiry
	}

	// Update the signal rule
	tflog.Debug(ctx, fmt.Sprintf("Update signal rule: %s", d.Id()), map[string]interface{}{
		"id":                             d.Id(),
		"notification_priority_override": d.Get("notification_priority_override"),
	})
	_, err := client.Sdk.Signals.UpdateTeamSignalRule(ctx, d.Get("team_id").(string), d.Id(), updateRequest)
	if err != nil {
		return diag.Errorf("Error updating signal rule %s: %v", d.Id(), err)
	}

	// Update state with the latest information from the API
	return readResourceFireHydrantSignalRule(ctx, d, m)
}

func deleteResourceFireHydrantSignalRule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Delete the signal rule
	tflog.Debug(ctx, fmt.Sprintf("Delete signal rule: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	err := client.Sdk.Signals.DeleteTeamSignalRule(ctx, d.Get("team_id").(string), d.Id())
	if err != nil {
		return diag.Errorf("Error deleting signal rule %s: %v", d.Id(), err)
	}

	return diag.Diagnostics{}
}
