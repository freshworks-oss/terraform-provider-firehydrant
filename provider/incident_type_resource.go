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
)

func resourceIncidentType() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceIncidentType,
		ReadContext:   readResourceIncidentType,
		UpdateContext: updateResourceIncidentType,
		DeleteContext: deleteResourceIncidentType,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"template": {
				Type:     schema.TypeList, // Using TypeList to simulate a map
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"customer_impact_summary": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"severity_slug": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"priority_slug": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"private_incident": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						// "labels": {
						// 	Type:     schema.TypeMap,
						// 	Optional: true,
						// },
						"tags": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"runbook_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"team_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"impacts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"impact_id": {
										Type:     schema.TypeString,
										Required: true,
									},
									"condition_id": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func createResourceIncidentType(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	description := d.Get("description").(string)
	templateDescription := d.Get("template.0.description").(string)
	cis := d.Get("template.0.customer_impact_summary").(string)
	severity_id := d.Get("template.0.severity_slug").(string)
	priority_id := d.Get("template.0.priority_slug").(string)
	//Seriously?!?  A pointer to a boolean?  The pointer takes up more space that the actual value.  Ugh.
	is_private := d.Get("template.0.private_incident").(bool)

	inputTags := d.Get("template.0.tags").([]interface{})
	tags := []string{}
	for _, tag := range inputTags {
		if v, ok := tag.(string); ok && v != "" {
			tags = append(tags, v)
		}
	}

	inputRunbooks := d.Get("template.0.runbook_ids").([]interface{})
	runbooks := []string{}
	for _, runbook := range inputRunbooks {
		if v, ok := runbook.(string); ok && v != "" {
			runbooks = append(runbooks, v)
		}
	}

	inputTeams := d.Get("template.0.team_ids").([]interface{})
	teams := []string{}
	for _, team := range inputTeams {
		if v, ok := team.(string); ok && v != "" {
			teams = append(teams, v)
		}
	}

	inputImpacts := d.Get("template.0.impacts").([]interface{})
	impacts := []components.CreateIncidentTypeImpact{}
	for _, impact := range inputImpacts {
		impactMap := impact.(map[string]interface{})
		impacts = append(impacts, components.CreateIncidentTypeImpact{
			ID:          impactMap["impact_id"].(string),
			ConditionID: impactMap["condition_id"].(string),
		})
	}

	request := components.CreateIncidentType{
		Name:        d.Get("name").(string),
		Description: &description,
		Template: components.CreateIncidentTypeTemplate{
			Description:           &templateDescription,
			CustomerImpactSummary: &cis,
			Severity:              &severity_id,
			Priority:              &priority_id,
			PrivateIncident:       &is_private,
			TagList:               tags,
			RunbookIds:            runbooks,
			TeamIds:               teams,
			Impacts:               impacts,
		},
	}

	tflog.Debug(ctx, "Create new Incident Type")
	response, err := client.Sdk.IncidentSettings.CreateIncidentType(ctx, request)
	if err != nil {
		return diag.Errorf("Error creating new Incident Type: %v", err)
	}

	responseID := response.GetID()
	if responseID == nil || *responseID == "" {
		return diag.Errorf("Error creating new Incident Type: the API returned an incident type with no ID")
	}
	d.SetId(*responseID)

	return readResourceIncidentType(ctx, d, m)
}

func readResourceIncidentType(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	id := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read incident type: %s", id), map[string]interface{}{
		"id": id,
	})

	response, err := client.Sdk.IncidentSettings.GetIncidentType(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	// labels is in the sdk as an empty struct, which seems... wrong.  I'm going to implement the rest of this without it
	// (because I can only hold so much complexity in my head), and then investigate this from the API side to see if
	// this is being generated correctly.

	templateSlice, diags := incidentTypeTemplateToState(id, response.GetTemplate())

	attributes := map[string]interface{}{
		"name":        stringValue(response.GetName()),
		"description": stringValue(response.GetDescription()),
		"template":    templateSlice,
	}

	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for incident_type %s: %v", key, id, err)
		}
	}

	responseID := response.GetID()
	if responseID == nil || *responseID == "" {
		return diag.Errorf("Error reading incident type %s: the API returned an incident type with no ID", id)
	}
	d.SetId(*responseID)

	return diags
}

// incidentTypeTemplateToState flattens an incident type's template into the shape
// Terraform expects.
//
// Incident types created before templates gained their current fields come back
// with the template object, or individual fields within it, missing. Every value is
// therefore read through the SDK's nil-safe accessors rather than dereferenced, and
// an absent template yields a template of zero values plus a warning, because the
// schema requires the block to be present.
func incidentTypeTemplateToState(id string, template *components.NullableIncidentTypeEntityTemplateEntity) ([]map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if template == nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Incident type has no template",
			Detail: fmt.Sprintf(
				"FireHydrant returned incident type %s without a template, so every template "+
					"attribute is read as empty. This is expected for incident types that predate "+
					"incident type templates.", id),
		})
	}

	// An impact missing either ID cannot be expressed in the schema, where both are
	// required, so it is skipped rather than written as an empty string.
	impacts := make([]map[string]interface{}, 0, len(template.GetImpacts()))
	skippedImpacts := 0
	for _, impact := range template.GetImpacts() {
		impactID, conditionID := impact.GetID(), impact.GetConditionID()
		if impactID == nil || *impactID == "" || conditionID == nil || *conditionID == "" {
			skippedImpacts++
			continue
		}
		impacts = append(impacts, map[string]interface{}{
			"impact_id":    *impactID,
			"condition_id": *conditionID,
		})
	}
	if skippedImpacts > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Incident type has incomplete impacts",
			Detail: fmt.Sprintf(
				"Incident type %s has %d impact(s) missing an impact ID or condition ID. They are "+
					"omitted from state, and applying a change to the template will drop them.",
				id, skippedImpacts),
		})
	}

	return []map[string]interface{}{{
		"description":             stringValue(template.GetDescription()),
		"customer_impact_summary": stringValue(template.GetCustomerImpactSummary()),
		"severity_slug":           stringValue(template.GetSeverity()),
		"priority_slug":           stringValue(template.GetPriority()),
		"private_incident":        boolValue(template.GetPrivateIncident()),
		"tags":                    stringsToList(template.GetTagList()),
		"runbook_ids":             stringsToList(template.GetRunbookIds()),
		"team_ids":                stringsToList(template.GetTeamIds()),
		"impacts":                 impacts,
	}}, diags
}

func updateResourceIncidentType(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	id := d.Id()
	description := d.Get("description").(string)
	templateDescription := d.Get("template.0.description").(string)
	cis := d.Get("template.0.customer_impact_summary").(string)
	severity_id := d.Get("template.0.severity_slug").(string)
	priority_id := d.Get("template.0.priority_slug").(string)
	//Seriously?!?  A pointer to a boolean?  The pointer takes up more space that the actual value.  Ugh.
	is_private := d.Get("template.0.private_incident").(bool)

	inputTags := d.Get("template.0.tags").([]interface{})
	tags := []string{}
	for _, tag := range inputTags {
		if v, ok := tag.(string); ok && v != "" {
			tags = append(tags, v)
		}
	}

	inputRunbooks := d.Get("template.0.runbook_ids").([]interface{})
	runbooks := []string{}
	for _, runbook := range inputRunbooks {
		if v, ok := runbook.(string); ok && v != "" {
			runbooks = append(runbooks, v)
		}
	}

	inputTeams := d.Get("template.0.team_ids").([]interface{})
	teams := []string{}
	for _, team := range inputTeams {
		if v, ok := team.(string); ok && v != "" {
			teams = append(teams, v)
		}
	}

	inputImpacts := d.Get("template.0.impacts").([]interface{})
	impacts := []components.UpdateIncidentTypeImpact{}
	for _, impact := range inputImpacts {
		impactMap := impact.(map[string]interface{})
		impacts = append(impacts, components.UpdateIncidentTypeImpact{
			ID:          impactMap["impact_id"].(string),
			ConditionID: impactMap["condition_id"].(string),
		})
	}

	request := components.UpdateIncidentType{
		Name:        d.Get("name").(string),
		Description: &description,
		Template: components.UpdateIncidentTypeTemplate{
			Description:           &templateDescription,
			CustomerImpactSummary: &cis,
			Severity:              &severity_id,
			Priority:              &priority_id,
			PrivateIncident:       &is_private,
			TagList:               tags,
			RunbookIds:            runbooks,
			TeamIds:               teams,
			Impacts:               impacts,
		},
	}

	tflog.Debug(ctx, "Update Incident Type")
	_, err := client.Sdk.IncidentSettings.UpdateIncidentType(ctx, id, request)
	if err != nil {
		return diag.Errorf("Error updating Incident Type: %v", err)
	}

	return readResourceIncidentType(ctx, d, m)
}

func deleteResourceIncidentType(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*firehydrant.APIClient)

	id := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Delete incident type: %s", id), map[string]interface{}{
		"ID": id,
	})
	err := client.Sdk.IncidentSettings.DeleteIncidentType(ctx, id)
	if err != nil {
		if err.(*sdkerrors.SDKError).StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error deleting incident type %s: %v", id, err)
	}

	return diag.Diagnostics{}
}
