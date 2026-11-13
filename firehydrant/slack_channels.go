package firehydrant

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type SlackChannelParams struct {
	ID   string
	Name string
}

// SlackChannelsClient is an interface for interacting with Slack channels
type SlackChannelsClient interface {
	Get(ctx context.Context, params SlackChannelParams) (*SlackChannelResponse, error)
}

// RESTSlackChannelsClient implements the SlackChannelClient interface
type RESTSlackChannelsClient struct {
	client *APIClient
}

var _ SlackChannelsClient = &RESTSlackChannelsClient{}

// Get retrieves a Slack channel from FireHydrant using Slack ID. This is useful for looking up
// a Slack channel's internal ID.
func (c *RESTSlackChannelsClient) Get(ctx context.Context, params SlackChannelParams) (*SlackChannelResponse, error) {
	channels := &SlackChannelsResponse{}

	query := ""
	if params.ID != "" {
		query = fmt.Sprintf("slack_channel_id=%s", params.ID)
	} else if params.Name != "" {
		query = fmt.Sprintf("name=%s", strings.TrimPrefix(params.Name, "#"))
	}

	if err := c.client.LegacyGet(ctx, "integrations/slack/channels?"+query, nil, channels); err != nil {
		return nil, fmt.Errorf("could not get slack channel: %w", err)
	}

	if len(channels.Channels) == 0 { //len of nil slices is defined as 0
		return nil, fmt.Errorf("no slack channel found with options name '%s' and / or id '%s'", params.Name, params.ID)
	}
	if channelCount := len(channels.Channels); channelCount > 1 {
		// "at least" because it may paginate.
		tflog.Error(ctx, "found more than one Slack channel", map[string]interface{}{
			"id":    params.ID,
			"name":  params.Name,
			"found": channelCount,
		})
		for _, channel := range channels.Channels {
			tflog.Error(ctx, "found Slack channel", map[string]interface{}{
				"id":                 params.ID,
				"name":               params.Name,
				"slack_channel_id":   channel.SlackChannelID,
				"slack_channel_name": channel.Name,
			})
		}
		return nil, fmt.Errorf("more than one Slack channel found: see Terraform logs for more information")
	}

	tflog.Info(ctx, "found Slack channel", map[string]interface{}{
		"id":                 channels.Channels[0].ID,
		"name":               channels.Channels[0].Name,
		"slack_channel_id":   channels.Channels[0].SlackChannelID,
		"slack_channel_name": channels.Channels[0].Name,
	})

	return channels.Channels[0], nil
}
