package firehydrant

import (
	"context"
	"fmt"
	"time"
)

// RunbooksClient is an interface for interacting with runbooks on FireHydrant
type RunbookActionsClient interface {
	Get(ctx context.Context, runbookType string, integrationSlug string, actionSlug string) (*RunbookAction, error)
}

// RESTRunbooksClient implements the RunbooksClient interface
type RESTRunbookActionsClient struct {
	client *APIClient
}

var _ RunbookActionsClient = &RESTRunbookActionsClient{}

// RunbookActionsResponse is the payload for retrieving runbook actions
// URL: GET https://api.firehydrant.io/v1/runbooks/actions
type RunbookActionsResponse struct {
	Actions []RunbookAction `json:"data"`
}

type RunbookAction struct {
	ID          string       `json:"id"`
	Integration *Integration `json:"integration"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Integration struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type RunbookActionsQuery struct {
	Type  string `url:"type,omitempty"`
	Items uint   `url:"per_page,omitempty"`
	Lite  bool   `url:"lite,omitempty"`
}

// Get returns a runbook action from the FireHydrant API
func (c *RESTRunbookActionsClient) Get(ctx context.Context, runbookType string, integrationSlug string, actionSlug string) (*RunbookAction, error) {
	runbookActionResponse := &RunbookActionsResponse{}
	query := RunbookActionsQuery{Type: runbookType, Items: 100, Lite: true}

	if err := c.client.LegacyGet(ctx, "runbooks/actions", query, runbookActionResponse); err != nil {
		return nil, fmt.Errorf("could not get runbook actions: %w", err)
	}

	for _, action := range runbookActionResponse.Actions {
		if action.Slug == actionSlug && action.Integration.Slug == integrationSlug {
			return &action, nil
		}
	}

	return nil, fmt.Errorf("could not find runbook action")
}
