package firehydrant

import (
	"context"
	"fmt"
)

// SeverityType represents the type of the severity.
type SeverityType string

// List of valid severity types
const (
	SeverityTypeGameday            SeverityType = "gameday"
	SeverityTypeMaintenance        SeverityType = "maintenance"
	SeverityTypeUnexpectedDowntime SeverityType = "unexpected_downtime"
)

// SeveritiesClient is an interface for interacting with severities on FireHydrant
type SeveritiesClient interface {
	Get(ctx context.Context, slug string) (*SeverityResponse, error)
	Create(ctx context.Context, createReq CreateSeverityRequest) (*SeverityResponse, error)
	Update(ctx context.Context, slug string, updateReq UpdateSeverityRequest) (*SeverityResponse, error)
	Delete(ctx context.Context, slug string) error
}

// RESTSeveritiesClient implements the SeveritiesClient interface
type RESTSeveritiesClient struct {
	client *APIClient
}

var _ SeveritiesClient = &RESTSeveritiesClient{}

// SeverityResponse is the payload for a single environment
// URL: GET https://api.firehydrant.io/v1/severities/{id}
type SeverityResponse struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Get retrieves a severity from FireHydrant
func (c *RESTSeveritiesClient) Get(ctx context.Context, slug string) (*SeverityResponse, error) {
	sevResponse := &SeverityResponse{}

	if err := c.client.LegacyGet(ctx, "severities/"+slug, nil, sevResponse); err != nil {
		return nil, fmt.Errorf("could not get severity: %w", err)
	}

	return sevResponse, nil
}

// CreateSeverityRequest is the payload for creating a severity
// URL: POST https://api.firehydrant.io/v1/severities
type CreateSeverityRequest struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Create creates a severity
func (c *RESTSeveritiesClient) Create(ctx context.Context, createReq CreateSeverityRequest) (*SeverityResponse, error) {
	sevResponse := &SeverityResponse{}
	if err := c.client.LegacyPost(ctx, "severities", &createReq, sevResponse); err != nil {
		return nil, fmt.Errorf("could not create severity: %w", err)
	}

	return sevResponse, nil
}

// UpdateSeverityRequest is the payload for updating a severity
// URL: PATCH https://api.firehydrant.io/v1/severities/{id}
type UpdateSeverityRequest struct {
	Slug        string `json:"slug,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Update updates a severity in FireHydrant
func (c *RESTSeveritiesClient) Update(ctx context.Context, slug string, updateReq UpdateSeverityRequest) (*SeverityResponse, error) {
	sevResponse := &SeverityResponse{}
	if err := c.client.LegacyPatch(ctx, "severities/"+slug, &updateReq, sevResponse); err != nil {
		return nil, fmt.Errorf("could not update severity: %w", err)
	}

	return sevResponse, nil
}

// Delete deletes a severity from FireHydrant
func (c *RESTSeveritiesClient) Delete(ctx context.Context, slug string) error {
	err := c.client.LegacyDelete(ctx, "severities/"+slug)
	if err != nil {
		return fmt.Errorf("could not delete severity: %w", err)
	}

	return nil
}
