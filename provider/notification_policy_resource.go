package provider

import (
	"context"
	"fmt"

	"github.com/firehydrant/firehydrant-go-sdk/models/operations"
	"github.com/firehydrant/firehydrant-go-sdk/models/sdkerrors"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceNotificationPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceFireHydrantNotificationPolicy,
		UpdateContext: updateResourceFireHydrantNotificationPolicy,
		ReadContext:   readResourceFireHydrantNotificationPolicy,
		DeleteContext: deleteResourceFireHydrantNotificationPolicy,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			// Required
			"notification_group_method": {
				Type:     schema.TypeString,
				Required: true,
			},
			"max_delay": {
				Type:     schema.TypeString,
				Required: true,
			},
			"priority": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func readResourceFireHydrantNotificationPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	notificationPolicyID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read notification policy: %s", notificationPolicyID), map[string]interface{}{
		"id": notificationPolicyID,
	})

	response, err := client.Sdk.Signals.GetNotificationPolicy(ctx, notificationPolicyID)
	if err != nil {
		return diag.Errorf("Error reading notification policy %s: %v", notificationPolicyID, err)
	}

	attributes := map[string]interface{}{
		"notification_group_method": string(*response.NotificationGroupMethod),
		"max_delay":                 *response.GetMaxDelay(),
		"priority":                  string(*response.Priority),
	}

	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for notification_policy %s: %v", key, notificationPolicyID, err)
		}
	}

	return diag.Diagnostics{}
}

func createResourceFireHydrantNotificationPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	var ngm operations.CreateNotificationPolicyNotificationGroupMethod
	desiredNgm := d.Get("notification_group_method").(string)
	switch desiredNgm {
	case "any":
		fallthrough
	case "push":
		fallthrough
	case "email":
		fallthrough
	case "voice":
		fallthrough
	case "mobile_text":
		fallthrough
	case "chat":
		ngm = operations.CreateNotificationPolicyNotificationGroupMethod(desiredNgm)
	default:
		return diag.Errorf("invalid value for notification_group_method: %v", desiredNgm)
	}

	maxDelay := d.Get("max_delay").(string)

	var priority operations.CreateNotificationPolicyPriority
	desiredPriority := d.Get("priority").(string)
	switch desiredPriority {
	case "HIGH":
		fallthrough
	case "MEDIUM":
		fallthrough
	case "LOW":
		priority = operations.CreateNotificationPolicyPriority(desiredPriority)
	default:
		return diag.Errorf("invalid value for priority: %v", desiredPriority)
	}

	createRequest := operations.CreateNotificationPolicyRequest{
		NotificationGroupMethod: ngm,
		MaxDelay:                maxDelay,
		Priority:                priority,
	}

	tflog.Debug(ctx, "Create new notification policy")
	serviceResponse, err := client.Sdk.Signals.CreateNotificationPolicy(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating new notification policy: %v", err)
	}

	// Set the new service's ID in state
	d.SetId(*serviceResponse.ID)

	// Update state with the latest information from the API
	return readResourceFireHydrantNotificationPolicy(ctx, d, m)
}

func updateResourceFireHydrantNotificationPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	var ngm operations.UpdateNotificationPolicyNotificationGroupMethod
	desiredNgm := d.Get("notification_group_method").(string)
	switch desiredNgm {
	case "any":
		fallthrough
	case "push":
		fallthrough
	case "email":
		fallthrough
	case "voice":
		fallthrough
	case "mobile_text":
		fallthrough
	case "chat":
		ngm = operations.UpdateNotificationPolicyNotificationGroupMethod(desiredNgm)
	default:
		return diag.Errorf("invalid value for notification_group_method: %v", desiredNgm)
	}

	maxDelay := d.Get("max_delay").(string)

	var priority operations.UpdateNotificationPolicyPriority
	desiredPriority := d.Get("priority").(string)
	switch desiredPriority {
	case "HIGH":
		fallthrough
	case "MEDIUM":
		fallthrough
	case "LOW":
		priority = operations.UpdateNotificationPolicyPriority(desiredPriority)
	default:
		return diag.Errorf("invalid value for priority: %v", desiredPriority)
	}

	updateRequest := operations.UpdateNotificationPolicyRequestBody{
		NotificationGroupMethod: &ngm,
		MaxDelay:                &maxDelay,
		Priority:                &priority,
	}

	tflog.Debug(ctx, fmt.Sprintf("Update notification policy: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	err := client.Sdk.Signals.UpdateNotificationPolicy(ctx, d.Id(), &updateRequest)
	if err != nil {
		return diag.Errorf("Error updating notification policy %s: %v", d.Id(), err)
	}

	return readResourceFireHydrantNotificationPolicy(ctx, d, m)
}

func deleteResourceFireHydrantNotificationPolicy(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	notificationPolicyID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Delete notification policy: %s", notificationPolicyID), map[string]interface{}{
		"id": notificationPolicyID,
	})
	err := client.Sdk.Signals.DeleteNotificationPolicy(ctx, notificationPolicyID)
	if err != nil {
		if err.(*sdkerrors.SDKError).StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error deleting service %s: %v", notificationPolicyID, err)
	}

	return diag.Diagnostics{}
}
