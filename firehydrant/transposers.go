package firehydrant

import (
	"context"
	"fmt"
)

type TransposersParams struct {
	EscalationPolicyID string `url:"escalation_policy_id,omitempty"`
	OnCallScheduleID   string `url:"on_call_schedule_id,omitempty"`
	TeamID             string `url:"team_id,omitempty"`
	UserID             string `url:"user_id,omitempty"`
}

// IngestURLClient is an interface for interacting with ingest URLs
type TransposersClient interface {
	Get(ctx context.Context, params TransposersParams) (*TransposersResponse, error)
}

// RESTIngestURLClient implements the IngestURLClient interface
type RESTTransposersClient struct {
	client *APIClient
}

var _ TransposersClient = &RESTTransposersClient{}

// Get retrieves an ingest URL from FireHydrant.  See below for query params.
func (c *RESTTransposersClient) Get(ctx context.Context, params TransposersParams) (*TransposersResponse, error) {
	transposers := &TransposersResponse{}

	// The query here should only change the ingrest URL for each transposer.  It should be one of:
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

	if err := c.client.LegacyGet(ctx, "signals/transposers", params, transposers); err != nil {
		return nil, fmt.Errorf("could not get transposers: %w", err)
	}

	if len(transposers.Transposers) == 0 { //len of nil slices is defined as 0
		return nil, fmt.Errorf("no transposers found with options %#v", params)
	}

	return transposers, nil
}
