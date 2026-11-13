package firehydrant

import (
	"context"
	"fmt"
	"time"
)

// TaskListsClient is an interface for interacting with task lists on FireHydrant
type TaskListsClient interface {
	Get(ctx context.Context, id string) (*TaskListResponse, error)
	Create(ctx context.Context, createReq CreateTaskListRequest) (*TaskListResponse, error)
	Update(ctx context.Context, id string, updateReq UpdateTaskListRequest) (*TaskListResponse, error)
	Delete(ctx context.Context, id string) error
}

// RESTTaskListsClient implements the TaskListsClient interface
type RESTTaskListsClient struct {
	client *APIClient
}

var _ TaskListsClient = &RESTTaskListsClient{}

// TaskListResponse is the payload for retrieving a task list
// URL: GET https://api.firehydrant.io/v1/task_lists/{id}
type TaskListResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	TaskListItems []TaskListItem `json:"task_list_items"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskListItem is an item in a task list
type TaskListItem struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

// Get returns a task list from the FireHydrant API
func (c *RESTTaskListsClient) Get(ctx context.Context, id string) (*TaskListResponse, error) {
	taskListResponse := &TaskListResponse{}

	if err := c.client.LegacyGet(ctx, "task_lists/"+id, nil, taskListResponse); err != nil {
		return nil, fmt.Errorf("could not get task list: %w", err)
	}

	return taskListResponse, nil
}

// CreateTaskListRequest is the payload for creating a task list
// URL: POST https://api.firehydrant.io/v1/task_lists
type CreateTaskListRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	TaskListItems []TaskListItem `json:"task_list_items"`
}

// Create creates a brand spankin new task list in FireHydrant
func (c *RESTTaskListsClient) Create(ctx context.Context, createReq CreateTaskListRequest) (*TaskListResponse, error) {
	taskListResponse := &TaskListResponse{}
	if err := c.client.LegacyPost(ctx, "task_lists", &createReq, taskListResponse); err != nil {
		return nil, fmt.Errorf("could not create task list: %w", err)
	}

	return taskListResponse, nil
}

// UpdateTaskListRequest is the payload for updating a task list
// URL: PATCH https://api.firehydrant.io/v1/task_lists/{id}
type UpdateTaskListRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description"`

	TaskListItems []TaskListItem `json:"task_list_items"`
}

// Update updates a task list in FireHydrant
func (c *RESTTaskListsClient) Update(ctx context.Context, id string, updateReq UpdateTaskListRequest) (*TaskListResponse, error) {
	taskListResponse := &TaskListResponse{}
	if err := c.client.LegacyPatch(ctx, "task_lists/"+id, updateReq, taskListResponse); err != nil {
		return nil, fmt.Errorf("could not update task list: %w", err)
	}

	return taskListResponse, nil
}

func (c *RESTTaskListsClient) Delete(ctx context.Context, id string) error {
	err := c.client.LegacyDelete(ctx, "task_lists/"+id)
	if err != nil {
		return fmt.Errorf("could not delete task list: %w", err)
	}

	return nil
}
