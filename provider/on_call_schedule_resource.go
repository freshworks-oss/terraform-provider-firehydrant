package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	"github.com/firehydrant/firehydrant-go-sdk/models/sdkerrors"
	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOnCallSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceFireHydrantOnCallSchedule,
		ReadContext:   readResourceFireHydrantOnCallSchedule,
		UpdateContext: updateResourceFireHydrantOnCallSchedule,
		DeleteContext: deleteResourceFireHydrantOnCallSchedule,
		Importer: &schema.ResourceImporter{
			StateContext: importResourceFireHydrantOnCallSchedule,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"rotation_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: "Name of the schedule's primary rotation (the rotation FireHydrant " +
					"creates alongside the schedule itself). Set this to override the default " +
					"(which inherits the schedule's name). Tracked in state; mutations are sent " +
					"to the schedule's PATCH endpoint.",
			},
			"rotation_description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				Description: "Description of the schedule's primary rotation. Set this to override " +
					"the default (which inherits the schedule's description). Tracked in state.",
			},
			"member_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true, // will be required in the future once `members` has been removed.
				// Computed so that omitting the attribute means "FireHydrant owns membership"
				// rather than "membership is empty". An explicitly configured empty list is
				// still honored -- see memberIDsAreConfigured.
				Computed:      true,
				ConflictsWith: []string{"members"},
				Description: "IDs of the users in the schedule's rotation. Omit to leave membership " +
					"under FireHydrant's control. Cannot represent gap or unassigned slots -- use the " +
					"firehydrant_rotation resource for those.",
			},
			"members": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				// Technically, I (wilsonehusin) don't think this ever worked because it would produce HTTP 400s.
				// Documentation also always mentioned `member_ids` as the correct attribute to use.
				// Leaving this here for now to prevent potential breaking changes.
				Deprecated:    "Use member_ids to configure membership; members attribute will be removed in a future release.",
				ConflictsWith: []string{"member_ids"},
			},
			"time_zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"strategy": {
				Type:     schema.TypeList, // Using TypeList to simulate a map
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"handoff_time": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"handoff_day": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"shift_duration": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"start_time": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"color": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"slack_user_group_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"restrictions": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start_day": {
							Type:     schema.TypeString,
							Required: true,
						},
						"start_time": {
							Type:     schema.TypeString,
							Required: true,
						},
						"end_day": {
							Type:     schema.TypeString,
							Required: true,
						},
						"end_time": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"effective_at": {
				Type:     schema.TypeString,
				Optional: true,
				// Don't set computed:true since we don't want it in the state
				Description: "RFC3339 timestamp for when the schedule update should take effect. If not provided or if the time is in the past, the update will take effect immediately.",
				ValidateDiagFunc: schema.SchemaValidateDiagFunc(
					func(v interface{}, path cty.Path) diag.Diagnostics {
						timeStr := v.(string)
						_, err := time.Parse(time.RFC3339, timeStr)
						if err != nil {
							return diag.Diagnostics{
								diag.Diagnostic{
									Severity:      diag.Error,
									Summary:       "Invalid effective_at timestamp",
									Detail:        fmt.Sprintf("effective_at must be a valid RFC3339 timestamp (e.g. 2024-01-01T15:04:05Z), got: %s", timeStr),
									AttributePath: path,
								},
							}
						}

						return nil
					},
				),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return true
				},
			},
		},
	}
}

func createResourceFireHydrantOnCallSchedule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Create the on-call schedule
	teamID := d.Get("team_id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Create on-call schedule: %s", teamID), map[string]interface{}{
		"team_id": teamID,
	})

	inputMemberIDs := d.Get("member_ids").([]interface{})
	if len(inputMemberIDs) == 0 {
		inputMemberIDs = d.Get("members").([]interface{})
	}
	memberIDs := []string{}
	for _, memberID := range inputMemberIDs {
		if v, ok := memberID.(string); ok && v != "" {
			memberIDs = append(memberIDs, v)
		}
	}

	// Gather values from API response
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	timeZone := d.Get("time_zone").(string)
	startTime := d.Get("start_time").(string)
	handoffTime := d.Get("strategy.0.handoff_time").(string)
	handoffDay := d.Get("strategy.0.handoff_day").(string)
	shiftDuration := d.Get("strategy.0.shift_duration").(string)

	onCallSchedule := components.CreateTeamOnCallSchedule{
		Name:        name,
		Description: &description,
		TimeZone:    &timeZone,
		Strategy: &components.CreateTeamOnCallScheduleStrategy{
			Type:          components.CreateTeamOnCallScheduleType(d.Get("strategy.0.type").(string)),
			HandoffTime:   &handoffTime,
			HandoffDay:    (*components.CreateTeamOnCallScheduleHandoffDay)(&handoffDay),
			ShiftDuration: &shiftDuration,
		},
		MemberIds:    memberIDs,
		Restrictions: oncallRestrictionsFromDataSDK(d),
	}

	// start_time seeds the initial rotation's first shift for every strategy
	// type, not just custom. The API rejects an empty string, so only send it
	// when configured.
	if startTime != "" {
		onCallSchedule.StartTime = &startTime
	}

	// Get slack_user_group_id if set and non-empty
	if v, ok := d.GetOk("slack_user_group_id"); ok && v.(string) != "" {
		slackUserGroupID := v.(string)
		onCallSchedule.SlackUserGroupID = &slackUserGroupID
	}

	// Optional overrides for the schedule's initial rotation. FireHydrant always
	// creates one rotation alongside the schedule; without these, that rotation
	// inherits the schedule's name and description.
	if v, ok := d.GetOk("rotation_name"); ok && v.(string) != "" {
		rotationName := v.(string)
		onCallSchedule.RotationName = &rotationName
	}
	if v, ok := d.GetOk("rotation_description"); ok && v.(string) != "" {
		rotationDescription := v.(string)
		onCallSchedule.RotationDescription = &rotationDescription
	}

	if onCallSchedule.Strategy.Type != "" {
		isCustomStrategy := onCallSchedule.Strategy.Type == "custom"
		if isCustomStrategy {
			if onCallSchedule.Strategy.ShiftDuration == nil || *onCallSchedule.Strategy.ShiftDuration == "" {
				return diag.Errorf("firehydrant_on_call_schedule.strategy.shift_duration is required when strategy type is 'custom'")
			}
			if onCallSchedule.StartTime == nil || *onCallSchedule.StartTime == "" {
				return diag.Errorf("firehydrant_on_call_schedule.start_time is required when strategy type is 'custom'")
			}

			// Discard unused values to avoid ambiguity.
			onCallSchedule.Strategy.HandoffTime = nil
			onCallSchedule.Strategy.HandoffDay = nil
		} else {
			if onCallSchedule.Strategy.HandoffTime == nil || *onCallSchedule.Strategy.HandoffTime == "" {
				return diag.Errorf("firehydrant_on_call_schedule.strategy.handoff_time is required when strategy type is '%s'", onCallSchedule.Strategy.Type)
			}
			if onCallSchedule.Strategy.Type == "weekly" && (onCallSchedule.Strategy.HandoffDay == nil || *onCallSchedule.Strategy.HandoffDay == "") {
				return diag.Errorf("firehydrant_on_call_schedule.strategy.handoff_day is required when strategy type is '%s'", onCallSchedule.Strategy.Type)
			}

			// Discard unused values to avoid ambiguity.
			onCallSchedule.Strategy.ShiftDuration = nil
		}
	}

	// Create the on-call schedule
	createdOnCallSchedule, err := client.Sdk.Signals.CreateTeamOnCallSchedule(ctx, teamID, onCallSchedule)
	if err != nil {
		return diag.Errorf("Error creating on-call schedule %s: %v", teamID, err)
	}

	// Set the on-call schedule's ID in state
	d.SetId(*createdOnCallSchedule.GetID())

	return readResourceFireHydrantOnCallSchedule(ctx, d, m)
}

func readResourceFireHydrantOnCallSchedule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the on-call schedule
	id := d.Id()
	teamID := d.Get("team_id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Read on-call schedule: %s", id), map[string]interface{}{
		"id":      id,
		"team_id": teamID,
	})

	onCallSchedule, err := client.Sdk.Signals.GetTeamOnCallSchedule(ctx, teamID, id, nil, nil)
	if err != nil {
		if errors.Is(err, firehydrant.ErrorNotFound) {
			tflog.Debug(ctx, fmt.Sprintf("On-call schedule %s no longer exists", id), map[string]interface{}{
				"id":      id,
				"team_id": teamID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading on-call schedule %s: %v", id, err)
	}

	// Gather values from API response.
	//
	// A member with no ID is a gap or unassigned slot in the rotation. Those are
	// valid FireHydrant state but member_ids cannot represent them, so they are
	// skipped here and reported as a warning below.
	var diags diag.Diagnostics
	memberIDs := make([]string, 0, len(onCallSchedule.GetMembers()))
	skippedSlots := 0
	for _, member := range onCallSchedule.GetMembers() {
		if memberID := member.GetID(); memberID != nil && *memberID != "" {
			memberIDs = append(memberIDs, *memberID)
			continue
		}
		skippedSlots++
	}
	if skippedSlots > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "On-call schedule has slots that member_ids cannot represent",
			Detail: fmt.Sprintf(
				"Schedule %s has %d gap or unassigned slot(s) in its rotation. These are omitted from "+
					"member_ids, and applying a change to member_ids will delete them. Manage this "+
					"rotation with the firehydrant_rotation resource to preserve them.", id, skippedSlots),
		})
	}

	attributes := map[string]interface{}{
		"name":         *onCallSchedule.GetName(),
		"description":  *onCallSchedule.GetDescription(),
		"time_zone":    *onCallSchedule.GetTimeZone(),
		"member_ids":   memberIDs,
		"restrictions": restrictionsToDataSDK(onCallSchedule.GetRestrictions()),
	}

	// Handle strategy if it exists
	if strategy := onCallSchedule.GetStrategy(); strategy != nil {
		attributes["strategy"] = strategyToMapSDK(*strategy)
	}
	if slackUserGroupID := onCallSchedule.GetSlackUserGroupID(); slackUserGroupID != nil && *slackUserGroupID != "" {
		attributes["slack_user_group_id"] = *slackUserGroupID
	}

	if rotations := onCallSchedule.GetRotations(); len(rotations) > 0 {
		if name := rotations[0].GetName(); name != nil {
			attributes["rotation_name"] = *name
		}
		if description := rotations[0].GetDescription(); description != nil {
			attributes["rotation_description"] = *description
		}
	}

	// Set the data source attributes to the values we got from the API
	for key, val := range attributes {
		if err := d.Set(key, val); err != nil {
			return diag.Errorf("Error setting %s for on-call schedule %s: %v", key, id, err)
		}
	}

	// Set the on-call schedule's ID in state
	d.SetId(*onCallSchedule.GetID())

	return diags
}

// memberIDsAreConfigured reports whether the practitioner actually declared
// member_ids (or the deprecated members) in configuration. Because neither
// attribute can be distinguished from an empty list via d.Get, the raw config is
// consulted directly: a null value means the attribute is absent and FireHydrant
// owns membership, while an explicitly empty list means "remove all members".
//
// This matters because the API replaces the rotation's entire membership list
// whenever member_ids is present in the request body, including gap and
// unassigned slots that Terraform never saw.
func memberIDsAreConfigured(d *schema.ResourceData) bool {
	rawConfig := d.GetRawConfig()
	if rawConfig.IsNull() || !rawConfig.IsKnown() {
		return false
	}

	for _, attr := range []string{"member_ids", "members"} {
		v := rawConfig.GetAttr(attr)
		if !v.IsNull() && v.IsKnown() {
			return true
		}
	}

	return false
}

func updateResourceFireHydrantOnCallSchedule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	id := d.Id()
	teamID := d.Get("team_id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Update on-call schedule: %s", id), map[string]interface{}{
		"id":      id,
		"team_id": teamID,
	})

	// Initialize updateRequest with basic fields
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	updateRequest := components.UpdateTeamOnCallSchedule{
		Name:        &name,
		Description: &description,
	}

	if v, ok := d.GetOk("rotation_name"); ok && v.(string) != "" {
		rotationName := v.(string)
		updateRequest.RotationName = &rotationName
	}
	if v, ok := d.GetOk("rotation_description"); ok && v.(string) != "" {
		rotationDescription := v.(string)
		updateRequest.RotationDescription = &rotationDescription
	}

	// Get slack_user_group_id if set
	if v, ok := d.GetOk("slack_user_group_id"); ok {
		slackUserGroupID := v.(string)
		updateRequest.SlackUserGroupID = &slackUserGroupID
	}

	// Handle effective_at - always set it to ensure API gets a valid timestamp
	if raw := d.GetRawConfig().GetAttr("effective_at"); !raw.IsNull() {
		effectiveAtStr := raw.AsString()
		if effectiveAtStr != "" {
			// Validate the timestamp format
			_, err := time.Parse(time.RFC3339, effectiveAtStr)
			if err != nil {
				return diag.FromErr(err)
			}
			// Send the timestamp as-is to the API
			updateRequest.EffectiveAt = &effectiveAtStr
			tflog.Debug(ctx, "Schedule update will take effect at: "+effectiveAtStr, map[string]interface{}{
				"effective_at": effectiveAtStr,
			})
		} else {
			// If effective_at is provided but empty, use current time
			now := time.Now()
			effectiveAtStr := now.Format(time.RFC3339)
			updateRequest.EffectiveAt = &effectiveAtStr
			tflog.Debug(ctx, "effective_at is empty, using current time for immediate effect", map[string]interface{}{
				"current_time": effectiveAtStr,
			})
		}
	} else {
		// If effective_at is not provided at all, use current time for immediate effect
		now := time.Now()
		effectiveAtStr := now.Format(time.RFC3339)
		updateRequest.EffectiveAt = &effectiveAtStr
		tflog.Debug(ctx, "effective_at not provided, using current time for immediate effect", map[string]interface{}{
			"current_time": effectiveAtStr,
		})
	}

	// Get member IDs
	inputMemberIDs := d.Get("member_ids").([]interface{})
	if len(inputMemberIDs) == 0 {
		inputMemberIDs = d.Get("members").([]interface{})
	}
	memberIDs := []string{}
	for _, memberID := range inputMemberIDs {
		if v, ok := memberID.(string); ok && v != "" {
			memberIDs = append(memberIDs, v)
		}
	}
	// Only send member_ids when the practitioner declared it. Sending an empty
	// list makes the API replace the rotation's whole membership -- wiping real
	// users along with any gap or unassigned slots. See issue #243.
	if memberIDsAreConfigured(d) {
		updateRequest.MemberIds = memberIDs
	}

	// Get strategy configuration
	if v, ok := d.GetOk("strategy"); ok {
		if strategies := v.([]interface{}); len(strategies) > 0 {
			updateRequest.Strategy = buildUpdateStrategy(strategies[0].(map[string]interface{}))
		}
	}

	// Get restrictions
	restrictions := d.Get("restrictions").([]interface{})
	for _, r := range restrictions {
		restriction := r.(map[string]interface{})
		startDay := restriction["start_day"].(string)
		startTime := restriction["start_time"].(string)
		endDay := restriction["end_day"].(string)
		endTime := restriction["end_time"].(string)

		updateRequest.Restrictions = append(updateRequest.Restrictions, components.UpdateTeamOnCallScheduleRestriction{
			StartDay:  components.UpdateTeamOnCallScheduleStartDay(startDay),
			StartTime: startTime,
			EndDay:    components.UpdateTeamOnCallScheduleEndDay(endDay),
			EndTime:   endTime,
		})
	}

	// Update the on-call schedule
	_, err := client.Sdk.Signals.UpdateTeamOnCallSchedule(ctx, teamID, id, updateRequest)
	if err != nil {
		return diag.Errorf("Error updating on-call schedule %s: %v", id, err)
	}

	return readResourceFireHydrantOnCallSchedule(ctx, d, m)
}

// buildUpdateStrategy constructs the strategy payload for an on-call schedule update, discarding
// the fields that don't apply to the strategy type. handoff_day/handoff_time apply only to
// daily/weekly strategies; shift_duration applies only to custom. Updating a schedule (for
// example, reordering member_ids) re-sends the whole strategy block, and for a custom rotation the
// irrelevant handoff_day was rejected by the API with "strategy[handoff_day] does not have a valid
// value" (FH-3378). This mirrors the discard logic in the create path.
func buildUpdateStrategy(strategy map[string]interface{}) *components.UpdateTeamOnCallScheduleStrategy {
	strategyType := strategy["type"].(string)
	handoffTime := strategy["handoff_time"].(string)
	handoffDay := strategy["handoff_day"].(string)
	shiftDuration := strategy["shift_duration"].(string)

	built := &components.UpdateTeamOnCallScheduleStrategy{
		Type:          components.UpdateTeamOnCallScheduleType(strategyType),
		HandoffTime:   &handoffTime,
		HandoffDay:    (*components.UpdateTeamOnCallScheduleHandoffDay)(&handoffDay),
		ShiftDuration: &shiftDuration,
	}

	if strategyType == "custom" {
		built.HandoffTime = nil
		built.HandoffDay = nil
	} else {
		built.ShiftDuration = nil
	}

	return built
}

func deleteResourceFireHydrantOnCallSchedule(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	id := d.Id()
	teamID := d.Get("team_id").(string)
	tflog.Debug(ctx, fmt.Sprintf("Delete on-call schedule: %s", id), map[string]interface{}{
		"id":      id,
		"team_id": teamID,
	})

	// Delete the on-call schedule
	err := client.Sdk.Signals.DeleteTeamOnCallSchedule(ctx, teamID, id)
	if err != nil {
		// If the resource is already deleted (404), treat as success
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			// Resource already deleted, remove from state
			d.SetId("")
			return diag.Diagnostics{}
		}
		// If it's a server error during cleanup, check if resource was actually deleted
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode >= 500 {
			_, readErr := client.Sdk.Signals.GetTeamOnCallSchedule(ctx, teamID, id, nil, nil)
			if readErr != nil {
				// If read returns 404, the resource was actually deleted despite the 500 error
				if readSDKErr, readOk := readErr.(*sdkerrors.SDKError); readOk && readSDKErr.StatusCode == 404 {
					tflog.Warn(ctx, fmt.Sprintf("On-call schedule %s was deleted despite 500 error during cleanup", id))
					d.SetId("")
					return diag.Diagnostics{}
				}
				// If read returns another error, fall through to return the original delete error
			} else {
				// Resource still exists - the 500 error was real, return it
				return diag.Errorf("Error deleting on-call schedule %s: %v", id, err)
			}
		}
		return diag.Errorf("Error deleting on-call schedule %s: %v", id, err)
	}

	// Remove the on-call schedule's ID from state
	d.SetId("")

	return diag.Diagnostics{}
}

func importResourceFireHydrantOnCallSchedule(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	team_id, id, err := resourceFireHydrantOnCallScheduleParseId(d.Id())
	if err != nil {
		return nil, err
	}

	d.Set("team_id", team_id)
	d.SetId(id)

	return []*schema.ResourceData{d}, nil
}

func resourceFireHydrantOnCallScheduleParseId(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected Team_ID:Schedule_ID", id)
	}

	return parts[0], parts[1], nil
}

func strategyToMapSDK(strategy components.NullableSignalsAPIOnCallStrategyEntity) []map[string]interface{} {
	m := map[string]interface{}{"type": *strategy.GetType()}
	if *strategy.GetType() == "custom" {
		if shiftDuration := strategy.GetShiftDuration(); shiftDuration != nil {
			m["shift_duration"] = *shiftDuration
		}
	} else {
		if handoffTime := strategy.GetHandoffTime(); handoffTime != nil {
			m["handoff_time"] = *handoffTime
		}
	}
	if *strategy.GetType() == "weekly" {
		if handoffDay := strategy.GetHandoffDay(); handoffDay != nil {
			m["handoff_day"] = *handoffDay
		}
	}
	return []map[string]interface{}{m}
}

func oncallRestrictionsFromDataSDK(d *schema.ResourceData) []components.CreateTeamOnCallScheduleRestriction {
	restrictions := make([]components.CreateTeamOnCallScheduleRestriction, 0)
	for _, restriction := range d.Get("restrictions").([]interface{}) {
		restrictionMap := restriction.(map[string]interface{})
		startDay := restrictionMap["start_day"].(string)
		startTime := restrictionMap["start_time"].(string)
		endDay := restrictionMap["end_day"].(string)
		endTime := restrictionMap["end_time"].(string)

		restrictions = append(restrictions, components.CreateTeamOnCallScheduleRestriction{
			StartDay:  components.CreateTeamOnCallScheduleStartDay(startDay),
			StartTime: startTime,
			EndDay:    components.CreateTeamOnCallScheduleEndDay(endDay),
			EndTime:   endTime,
		})
	}
	return restrictions
}

func restrictionsToDataSDK(restrictions []components.SignalsAPIOnCallRestrictionEntity) []map[string]interface{} {
	restrictionMaps := make([]map[string]interface{}, 0)
	for _, restriction := range restrictions {
		restrictionMaps = append(restrictionMaps, map[string]interface{}{
			"start_day":  *restriction.GetStartDay(),
			"start_time": *restriction.GetStartTime(),
			"end_day":    *restriction.GetEndDay(),
			"end_time":   *restriction.GetEndTime(),
		})
	}
	return restrictionMaps
}
