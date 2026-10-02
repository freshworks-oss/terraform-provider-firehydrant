package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/firehydrant/terraform-provider-firehydrant/firehydrant"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	apiKeyName             = "api_key"
	firehydrantBaseURLName = "firehydrant_base_url"
)

// Provider returns a terraform provider for the FireHydrant API
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			apiKeyName: {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("FIREHYDRANT_API_KEY", nil),
			},
			firehydrantBaseURLName: {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("FIREHYDRANT_BASE_URL", "https://api.firehydrant.io/v1/"),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"firehydrant_environment":            resourceEnvironment(),
			"firehydrant_functionality":          resourceFunctionality(),
			"firehydrant_incident_role":          resourceIncidentRole(),
			"firehydrant_incident_type":          resourceIncidentType(),
			"firehydrant_lifecycle_milestone":    resourceLifecycleMilestone(),
			"firehydrant_priority":               resourcePriority(),
			"firehydrant_role":                   resourceRole(),
			"firehydrant_rotation":               resourceRotation(),
			"firehydrant_runbook":                resourceRunbook(),
			"firehydrant_service_dependency":     resourceServiceDependency(),
			"firehydrant_service":                resourceService(),
			"firehydrant_severity":               resourceSeverity(),
			"firehydrant_task_list":              resourceTaskList(),
			"firehydrant_team":                   resourceTeam(),
			"firehydrant_signal_rule":            resourceSignalRule(),
			"firehydrant_on_call_schedule":       resourceOnCallSchedule(),
			"firehydrant_escalation_policy":      resourceEscalationPolicy(),
			"firehydrant_status_update_template": resourceStatusUpdateTemplate(),
			"firehydrant_inbound_email":          resourceInboundEmail(),
			"firehydrant_custom_event_source":    resourceCustomEventSource(),
			"firehydrant_notification_policy":    resourceNotificationPolicy(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"firehydrant_environment":       dataSourceEnvironment(),
			"firehydrant_functionality":     dataSourceFunctionality(),
			"firehydrant_escalation_policy": dataSourceEscalationPolicy(),
			"firehydrant_incident_role":     dataSourceIncidentRole(),
			"firehydrant_incident_type":     dataSourceIncidentType(),
			"firehydrant_ingest_url":        dataSourceIngestURL(),
			"firehydrant_lifecycle_phase":   dataSourceLifecyclePhase(),
			"firehydrant_on_call_schedule":  dataSourceOnCallSchedule(),
			"firehydrant_on_call_schedules": dataSourceOnCallSchedules(),
			"firehydrant_priority":          dataSourcePriority(),
			"firehydrant_role":              dataSourceRole(),
			"firehydrant_rotation":          dataSourceRotation(),
			"firehydrant_runbook":           dataSourceRunbook(),
			"firehydrant_runbook_action":    dataSourceRunbookAction(),
			"firehydrant_schedule":          dataSourceSchedule(),
			"firehydrant_service":           dataSourceService(),
			"firehydrant_services":          dataSourceServices(),
			"firehydrant_severity":          dataSourceSeverity(),
			"firehydrant_signal_rule":       dataSourceSignalRule(),
			"firehydrant_slack_channel":     dataSourceSlackChannel(),
			"firehydrant_task_list":         dataSourceTaskList(),
			"firehydrant_team":              dataSourceTeam(),
			"firehydrant_teams":             dataSourceTeams(),
			"firehydrant_user":              dataSourceUser(),
			"firehydrant_permissions":       dataSourcePermissions(),
		},
	}

	provider.ConfigureContextFunc = func(ctx context.Context, rd *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := provider.TerraformVersion

		if terraformVersion == "" {
			terraformVersion = "unknown"
		}

		return setupFireHydrantContext(ctx, rd, terraformVersion)
	}

	return provider
}

func setupFireHydrantContext(ctx context.Context, rd *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	apiKey := rd.Get(apiKeyName).(string)
	fireHydrantBaseURL := rd.Get(firehydrantBaseURLName).(string)

	// Add minimal delay between provider initializations in CI to avoid rate limiting
	if os.Getenv("TF_ACC") != "" {
		time.Sleep(500 * time.Millisecond)
	}

	ac, err := firehydrant.NewRestClient(apiKey, firehydrant.WithBaseURL(fireHydrantBaseURL), firehydrant.WithUserAgentSuffix(fmt.Sprintf("terraform-%s", terraformVersion)))
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("could not initialize API client: %w", err))
	}

	_, err = ac.Ping(ctx)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// We're getting 429s during tests, so the var here is intended to reduce overall API calls.  The old provider, horrifyingly, seems
	// to do part of its setup during this Ping() call, so we won't disable that one, but just cutting the pings in half should be sufficient.
	if os.Getenv("TF_ACC") == "" {
		_, err = ac.Sdk.AccountSettings.Ping(ctx)
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	return ac, nil
}

func convertStringMap(sm map[string]interface{}) map[string]string {
	m := map[string]string{}
	for k, v := range sm {
		m[k] = v.(string)
	}

	return m
}
