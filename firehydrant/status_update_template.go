package firehydrant

import (
	"context"
	"fmt"
	"time"
)

type StatusUpdateTemplates interface {
	Get(ctx context.Context, id string) (*StatusUpdateTemplateResponse, error)
	Create(ctx context.Context, createReq CreateStatusUpdateTemplateRequest) (*StatusUpdateTemplateResponse, error)
	Update(ctx context.Context, id string, updateReq UpdateStatusUpdateTemplateRequest) (*StatusUpdateTemplateResponse, error)
	Delete(ctx context.Context, id string) error
}

type RESTStatusUpdateTemplateClient struct {
	client *APIClient
}

var _ StatusUpdateTemplates = &RESTStatusUpdateTemplateClient{}

type StatusUpdateTemplateResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Body string `json:"body"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateStatusUpdateTemplateRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type UpdateStatusUpdateTemplateRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

func (c *RESTStatusUpdateTemplateClient) Create(ctx context.Context, createReq CreateStatusUpdateTemplateRequest) (*StatusUpdateTemplateResponse, error) {
	statusUpdateTemplateResponse := &StatusUpdateTemplateResponse{}
	if err := c.client.LegacyPost(ctx, "status_update_templates", &createReq, statusUpdateTemplateResponse); err != nil {
		return nil, fmt.Errorf("could not create status update template: %w", err)
	}

	return statusUpdateTemplateResponse, nil
}

func (c *RESTStatusUpdateTemplateClient) Get(ctx context.Context, id string) (*StatusUpdateTemplateResponse, error) {
	statusUpdateTemplateResponse := &StatusUpdateTemplateResponse{}

	if err := c.client.LegacyGet(ctx, fmt.Sprintf("status_update_templates/%s", id), nil, statusUpdateTemplateResponse); err != nil {
		return nil, fmt.Errorf("could not get status update template: %w", err)
	}

	return statusUpdateTemplateResponse, nil
}

func (c *RESTStatusUpdateTemplateClient) Update(ctx context.Context, id string, updateReq UpdateStatusUpdateTemplateRequest) (*StatusUpdateTemplateResponse, error) {
	statusUpdateTemplateResponse := &StatusUpdateTemplateResponse{}
	if err := c.client.LegacyPatch(ctx, fmt.Sprintf("status_update_templates/%s", id), &updateReq, statusUpdateTemplateResponse); err != nil {
		return nil, fmt.Errorf("could not update status update template: %w", err)
	}

	return statusUpdateTemplateResponse, nil
}

func (c *RESTStatusUpdateTemplateClient) Delete(ctx context.Context, id string) error {
	err := c.client.LegacyDelete(ctx, fmt.Sprintf("status_update_templates/%s", id))
	if err != nil {
		return fmt.Errorf("could not delete status update template: %w", err)
	}

	return nil
}
