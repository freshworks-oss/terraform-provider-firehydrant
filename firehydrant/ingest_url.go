package firehydrant

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type IngestURLParams struct {
	EscalationPolicyID string `url:"escalation_policy_id,omitempty"`
	OnCallScheduleID   string `url:"on_call_schedule_id,omitempty"`
	TeamID             string `url:"team_id,omitempty"`
	UserID             string `url:"user_id,omitempty"`
}

// IngestURLClient is an interface for interacting with ingest URLs
type IngestURLClient interface {
	Get(ctx context.Context, params IngestURLParams) (*IngestURLResponse, error)
}

// RESTIngestURLClient implements the IngestURLClient interface
type RESTIngestURLClient struct {
	client *APIClient
}

var _ IngestURLClient = &RESTIngestURLClient{}

// Get retrieves an ingest URL from FireHydrant.  See below for query params.
func (c *RESTIngestURLClient) Get(ctx context.Context, params IngestURLParams) (*IngestURLResponse, error) {
	ingestURL := &IngestURLResponse{}

	// The query here should be one of:
	// - nothing (to get the default URL),
	// - a user_id only,
	// - a team_id only,
	// - an escalation_policy_id with corresponding team_id,
	// - a schedule_id with corresponding team_id
	if params.TeamID == "" && params.EscalationPolicyID != "" {
		return nil, fmt.Errorf("missing team_id for escalation_policy_id %s", params.EscalationPolicyID)
	}
	if params.TeamID == "" && params.OnCallScheduleID != "" {
		return nil, fmt.Errorf("missing team_id for on_call_schedule_id %s", params.OnCallScheduleID)
	}

	if err := c.client.LegacyGet(ctx, "signals/ingest_url", params, ingestURL); err != nil {
		return nil, fmt.Errorf("could not get ingest url: %w", err)
	}

	if ingestURL.URL == "" {
		return nil, fmt.Errorf("no ingest URL found with options %#v", params)
	}

	tflog.Info(ctx, "found ingest URL", map[string]interface{}{
		"url": ingestURL.URL,
	})

	return ingestURL, nil
}
