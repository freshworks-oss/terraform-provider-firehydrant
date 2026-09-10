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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceFunctionality() *schema.Resource {
	return &schema.Resource{
		CreateContext: createResourceFireHydrantFunctionality,
		UpdateContext: updateResourceFireHydrantFunctionality,
		ReadContext:   readResourceFireHydrantFunctionality,
		DeleteContext: deleteResourceFireHydrantFunctionality,
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
			"alert_on_add": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"auto_add_responding_team": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"external_resources": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"connection_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"remote_id": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"links": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"href_url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"icon_url": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"owner_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"service_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"service_tier": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      5,
				ValidateFunc: validation.IntBetween(0, 5),
			},
			"team_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func readResourceFireHydrantFunctionality(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get the functionality
	functionalityID := d.Id()
	tflog.Debug(ctx, fmt.Sprintf("Read functionality: %s", functionalityID), map[string]interface{}{
		"id": functionalityID,
	})
	functionalityResponse, err := client.Sdk.CatalogEntries.GetFunctionality(ctx, functionalityID)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			tflog.Debug(ctx, fmt.Sprintf("Functionality %s no longer exists", functionalityID), map[string]interface{}{
				"id": functionalityID,
			})
			d.SetId("")
			return nil
		}
		return diag.Errorf("Error reading functionality %s: %v", functionalityID, err)
	}

	// Ladder truck defines these types as `  expose :labels, documentation: {type: "object", desc: "An object of label key and values"} # rubocop:disable CustomCops/GrapeMissingType`
	// Previous implementation suggests these are always strings, adding Unmarshall into map[string]string to be defensive
	labelsMap, err := unmarshalLabels(functionalityResponse.Labels)
	if err != nil {
		return diag.Errorf("Error unmarshalling labels for functionality %s: %v", functionalityID, err)
	}

	description := ""
	if functionalityResponse.Description != nil {
		description = *functionalityResponse.Description
	}

	autoAddRespondingTeam := false
	if functionalityResponse.AutoAddRespondingTeam != nil {
		autoAddRespondingTeam = *functionalityResponse.AutoAddRespondingTeam
	}

	alertOnAdd := false
	if functionalityResponse.AlertOnAdd != nil {
		alertOnAdd = *functionalityResponse.AlertOnAdd
	}

	serviceTier := 5
	if functionalityResponse.ServiceTier != nil {
		serviceTier = *functionalityResponse.ServiceTier
	}

	// Gather values from API response
	attributes := map[string]interface{}{
		"name":                     *functionalityResponse.Name,
		"alert_on_add":             alertOnAdd,
		"description":              description,
		"auto_add_responding_team": autoAddRespondingTeam,
		"labels":                   labelsMap,
		"service_tier":             serviceTier,
	}

	links := make([]map[string]interface{}, 0, len(functionalityResponse.Links))
	for _, currentLink := range functionalityResponse.Links {
		link := map[string]interface{}{}
		if currentLink.HrefURL != nil {
			link["href_url"] = *currentLink.HrefURL
		}
		if currentLink.IconURL != nil {
			link["icon_url"] = *currentLink.IconURL
		}
		if currentLink.Name != nil {
			link["name"] = *currentLink.Name
		}
		links = append(links, link)
	}
	attributes["links"] = links

	// Process service IDs
	serviceIDs := make([]string, 0)
	for _, service := range functionalityResponse.Services {
		if service.ID != nil {
			serviceIDs = append(serviceIDs, *service.ID)
		}
	}
	attributes["service_ids"] = serviceIDs

	environmentIDs := make([]string, 0, len(functionalityResponse.Environments))
	for _, environment := range functionalityResponse.Environments {
		if environment.ID != nil {
			environmentIDs = append(environmentIDs, *environment.ID)
		}
	}
	attributes["environment_ids"] = environmentIDs

	externalResources := make([]map[string]interface{}, 0, len(functionalityResponse.ExternalResources))
	for _, currentExternalResource := range functionalityResponse.ExternalResources {
		externalResource := map[string]interface{}{}
		if currentExternalResource.ConnectionType != nil {
			externalResource["connection_type"] = *currentExternalResource.ConnectionType
		}
		if currentExternalResource.RemoteID != nil {
			externalResource["remote_id"] = *currentExternalResource.RemoteID
		}
		externalResources = append(externalResources, externalResource)
	}
	attributes["external_resources"] = externalResources

	// Process owner
	var ownerID string
	if functionalityResponse.Owner != nil && functionalityResponse.Owner.ID != nil {
		ownerID = *functionalityResponse.Owner.ID
	}
	attributes["owner_id"] = ownerID

	// Process team IDs
	var teamIDs []interface{}
	for _, team := range functionalityResponse.Teams {
		if team.ID != nil {
			teamIDs = append(teamIDs, *team.ID)
		}
	}
	attributes["team_ids"] = teamIDs

	// Set the resource attributes to the values we got from the API
	for key, value := range attributes {
		if err := d.Set(key, value); err != nil {
			return diag.Errorf("Error setting %s for functionality %s: %v", key, functionalityID, err)
		}
	}

	return diag.Diagnostics{}
}

func createResourceFireHydrantFunctionality(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Get attributes from config and construct the create request
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	alertOnAdd := d.Get("alert_on_add").(bool)
	autoAddRespondingTeam := d.Get("auto_add_responding_team").(bool)
	serviceTier := d.Get("service_tier").(int)
	labels := convertStringMap(d.Get("labels").(map[string]interface{}))

	createRequest := components.CreateFunctionality{
		Name:                  name,
		Description:           &description,
		Labels:                labels,
		ServiceTier:           (*components.CreateFunctionalityServiceTier)(&serviceTier),
		AlertOnAdd:            &alertOnAdd,
		AutoAddRespondingTeam: &autoAddRespondingTeam,
	}

	for _, currentLink := range d.Get("links").(*schema.Set).List() {
		link := currentLink.(map[string]interface{})
		createLink := components.CreateFunctionalityLink{
			HrefURL: link["href_url"].(string),
			Name:    link["name"].(string),
		}
		if iconURL, ok := link["icon_url"].(string); ok && iconURL != "" {
			createLink.IconURL = &iconURL
		}
		createRequest.Links = append(createRequest.Links, createLink)
	}

	// Process services
	serviceIDs := d.Get("service_ids").(*schema.Set).List()
	for _, serviceID := range serviceIDs {
		createRequest.Services = append(createRequest.Services, components.CreateFunctionalityService{
			ID: serviceID.(string),
		})
	}

	for _, environmentID := range d.Get("environment_ids").(*schema.Set).List() {
		createRequest.Environments = append(createRequest.Environments, components.CreateFunctionalityEnvironment{
			ID: environmentID.(string),
		})
	}

	for _, currentExternalResource := range d.Get("external_resources").(*schema.Set).List() {
		externalResource := currentExternalResource.(map[string]interface{})
		createExternalResource := components.CreateFunctionalityExternalResource{
			RemoteID: externalResource["remote_id"].(string),
		}
		if connectionType, ok := externalResource["connection_type"].(string); ok && connectionType != "" {
			createExternalResource.ConnectionType = &connectionType
		}
		createRequest.ExternalResources = append(createRequest.ExternalResources, createExternalResource)
	}

	// Process owner if set
	if ownerID, ok := d.GetOk("owner_id"); ok && ownerID.(string) != "" {
		createRequest.Owner = &components.CreateFunctionalityOwner{
			ID: ownerID.(string),
		}
	}

	// Process team IDs if set
	teamIDs := d.Get("team_ids").(*schema.Set).List()
	for _, teamID := range teamIDs {
		createRequest.Teams = append(createRequest.Teams, components.CreateFunctionalityTeam{
			ID: teamID.(string),
		})
	}

	// Create the new functionality
	tflog.Debug(ctx, fmt.Sprintf("Create functionality: %s", name), map[string]interface{}{
		"name": name,
	})
	functionalityResponse, err := client.Sdk.CatalogEntries.CreateFunctionality(ctx, createRequest)
	if err != nil {
		return diag.Errorf("Error creating functionality %s: %v", name, err)
	}

	// Set the new functionality's ID in state
	d.SetId(*functionalityResponse.ID)

	// Update state with the latest information from the API
	return readResourceFireHydrantFunctionality(ctx, d, m)
}

func updateResourceFireHydrantFunctionality(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Construct the update request
	name := d.Get("name").(string)
	description := d.Get("description").(string)
	alertOnAdd := d.Get("alert_on_add").(bool)
	autoAddRespondingTeam := d.Get("auto_add_responding_team").(bool)
	serviceTier := d.Get("service_tier").(int)
	labels := convertStringMap(d.Get("labels").(map[string]interface{}))

	removeRemainingServices := true
	updateRequest := components.UpdateFunctionality{
		Name:                    &name,
		Description:             &description,
		Labels:                  labels,
		ServiceTier:             (*components.UpdateFunctionalityServiceTier)(&serviceTier),
		AlertOnAdd:              &alertOnAdd,
		AutoAddRespondingTeam:   &autoAddRespondingTeam,
		RemoveRemainingServices: &removeRemainingServices,
	}

	updateRequest.Links = []components.UpdateFunctionalityLink{}
	for _, currentLink := range d.Get("links").(*schema.Set).List() {
		link := currentLink.(map[string]interface{})
		updateLink := components.UpdateFunctionalityLink{
			HrefURL: link["href_url"].(string),
			Name:    link["name"].(string),
		}
		if iconURL, ok := link["icon_url"].(string); ok && iconURL != "" {
			updateLink.IconURL = &iconURL
		}
		updateRequest.Links = append(updateRequest.Links, updateLink)
	}

	// Process services
	// Always initialize Services as empty slice (not nil) so empty array is sent to clear services
	updateRequest.Services = []components.UpdateFunctionalityService{}
	serviceIDs := d.Get("service_ids").(*schema.Set).List()
	for _, serviceID := range serviceIDs {
		updateRequest.Services = append(updateRequest.Services, components.UpdateFunctionalityService{
			ID: serviceID.(string),
		})
	}

	updateRequest.Environments = []components.UpdateFunctionalityEnvironment{}
	for _, environmentID := range d.Get("environment_ids").(*schema.Set).List() {
		updateRequest.Environments = append(updateRequest.Environments, components.UpdateFunctionalityEnvironment{
			ID: environmentID.(string),
		})
	}
	removeRemainingEnvironments := true
	updateRequest.RemoveRemainingEnvironments = &removeRemainingEnvironments

	updateRequest.ExternalResources = []components.UpdateFunctionalityExternalResource{}
	for _, currentExternalResource := range d.Get("external_resources").(*schema.Set).List() {
		externalResource := currentExternalResource.(map[string]interface{})
		updateExternalResource := components.UpdateFunctionalityExternalResource{
			RemoteID: externalResource["remote_id"].(string),
		}
		if connectionType, ok := externalResource["connection_type"].(string); ok && connectionType != "" {
			updateExternalResource.ConnectionType = &connectionType
		}
		updateRequest.ExternalResources = append(updateRequest.ExternalResources, updateExternalResource)
	}
	removeRemainingExternalResources := true
	updateRequest.RemoveRemainingExternalResources = &removeRemainingExternalResources

	// Process owner - set or remove
	ownerID, ownerIDSet := d.GetOk("owner_id")
	if ownerIDSet && ownerID.(string) != "" {
		updateRequest.Owner = &components.UpdateFunctionalityOwner{
			ID: ownerID.(string),
		}
	} else {
		removeOwner := true
		updateRequest.RemoveOwner = &removeOwner
	}

	// Process team IDs
	// Always initialize Teams as empty slice (not nil) so empty array is sent to clear teams
	updateRequest.Teams = []components.UpdateFunctionalityTeam{}
	teamIDs := d.Get("team_ids").(*schema.Set).List()
	for _, teamID := range teamIDs {
		updateRequest.Teams = append(updateRequest.Teams, components.UpdateFunctionalityTeam{
			ID: teamID.(string),
		})
	}
	// Force replacement of teams with the ones we send
	removeRemainingTeams := true
	updateRequest.RemoveRemainingTeams = &removeRemainingTeams

	// Update the functionality
	tflog.Debug(ctx, fmt.Sprintf("Update functionality: %s", d.Id()), map[string]interface{}{
		"id": d.Id(),
	})
	_, err := client.Sdk.CatalogEntries.UpdateFunctionality(ctx, d.Id(), updateRequest)
	if err != nil {
		return diag.Errorf("Error updating functionality %s: %v", d.Id(), err)
	}

	// Update state with the latest information from the API
	return readResourceFireHydrantFunctionality(ctx, d, m)
}

func deleteResourceFireHydrantFunctionality(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Get the API client
	client := m.(*firehydrant.APIClient)

	// Delete the functionality
	functionalityID := d.Id()
	err := client.Sdk.CatalogEntries.DeleteFunctionality(ctx, functionalityID)
	if err != nil {
		if sdkErr, ok := err.(*sdkerrors.SDKError); ok && sdkErr.StatusCode == 404 {
			return nil
		}
		return diag.Errorf("Error deleting functionality %s: %v", functionalityID, err)
	}

	return diag.Diagnostics{}
}
