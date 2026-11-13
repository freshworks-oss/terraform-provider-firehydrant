package firehydrant

import (
	"context"
	"fmt"
	"time"
)

// ServiceDependenciesClient is an interface for interacting with service dependencies on FireHydrant
type ServiceDependenciesClient interface {
	Get(ctx context.Context, id string) (*ServiceDependencyResponse, error)
	Create(ctx context.Context, createReq CreateServiceDependencyRequest) (*ServiceDependencyResponse, error)
	Update(ctx context.Context, id string, updateReq UpdateServiceDependencyRequest) (*ServiceDependencyResponse, error)
	Delete(ctx context.Context, id string) error
}

// RESTServiceDependenciesClient implements the ServiceDependenciesClient interface
type RESTServiceDependenciesClient struct {
	client *APIClient
}

var _ ServiceDependenciesClient = &RESTServiceDependenciesClient{}

// ServiceDependencyResponse is the payload for retrieving a service dependency
// URL: GET https://api.firehydrant.io/v1/service_dependencies/{id}
type ServiceDependencyResponse struct {
	ID    string `json:"id"`
	Notes string `json:"notes"`

	ConnectedService *ServiceDependencyService `json:"connected_service"`
	Service          *ServiceDependencyService `json:"service"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ServiceDependencyService is a service in a service dependency
type ServiceDependencyService struct {
	ID string `json:"id"`
}

// Get returns a service dependency from the FireHydrant API
func (c *RESTServiceDependenciesClient) Get(ctx context.Context, id string) (*ServiceDependencyResponse, error) {
	serviceDependencyResponse := &ServiceDependencyResponse{}

	if err := c.client.LegacyGet(ctx, "service_dependencies/"+id, nil, serviceDependencyResponse); err != nil {
		return nil, fmt.Errorf("could not get service dependency: %w", err)
	}

	return serviceDependencyResponse, nil
}

// CreateServiceDependencyRequest is the payload for creating a service dependency
// URL: POST https://api.firehydrant.io/v1/service_dependencies
type CreateServiceDependencyRequest struct {
	ConnectedServiceID string `json:"connected_service_id"`
	ServiceID          string `json:"service_id"`
	Notes              string `json:"notes,omitempty"`
}

// Create creates a brand spankin new service dependency in FireHydrant
func (c *RESTServiceDependenciesClient) Create(ctx context.Context, createReq CreateServiceDependencyRequest) (*ServiceDependencyResponse, error) {
	serviceDependencyResponse := &ServiceDependencyResponse{}
	if err := c.client.LegacyPost(ctx, "service_dependencies", &createReq, serviceDependencyResponse); err != nil {
		return nil, fmt.Errorf("could not create service dependency: %w", err)
	}

	return serviceDependencyResponse, nil
}

// UpdateServiceDependencyRequest is the payload for updating a service dependency
// URL: PATCH https://api.firehydrant.io/v1/service_dependencies/{id}
type UpdateServiceDependencyRequest struct {
	Notes string `json:"notes"`
}

// Update updates a service dependency in FireHydrant
func (c *RESTServiceDependenciesClient) Update(ctx context.Context, id string, updateReq UpdateServiceDependencyRequest) (*ServiceDependencyResponse, error) {
	serviceDependencyResponse := &ServiceDependencyResponse{}
	if err := c.client.LegacyPatch(ctx, "service_dependencies/"+id, updateReq, serviceDependencyResponse); err != nil {
		return nil, fmt.Errorf("could not update service dependency: %w", err)
	}

	return serviceDependencyResponse, nil
}

func (c *RESTServiceDependenciesClient) Delete(ctx context.Context, id string) error {
	err := c.client.LegacyDelete(ctx, "service_dependencies/"+id)
	if err != nil {
		return fmt.Errorf("could not delete service dependency: %w", err)
	}

	return nil
}
