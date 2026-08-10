package firehydrant

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"

	"github.com/dghubble/sling"
	"github.com/pkg/errors"
)

const (
	// UserAgentPrefix is the prefix of the User-Agent header that all terraform REST calls perform
	UserAgentPrefix = "firehydrant-terraform-provider"
)

// checkResponseStatusCode checks to see if the response's status
// code corresponds to an error or not. An error is returned for
// all status codes 300 and above
func checkResponseStatusCode(response *http.Response, apiError *APIError) error {
	switch code := response.StatusCode; {
	case code >= 200 && code <= 299:
		return nil
	case code == 404:
		req := response.Request
		return fmt.Errorf("%w: %s '%s'", ErrorNotFound, req.Method, req.URL.String())
	case code == 401:
		return fmt.Errorf("%s\n%s", ErrorUnauthorized, apiError)
	default:
		return fmt.Errorf("%d request failed with error\n%s", code, apiError)
	}
}

// APIClient is the client that accesses all of the api.firehydrant.io resources
type APIClient struct {
	baseURL         string
	token           string
	userAgentSuffix string

	Sdk *fhsdk.FireHydrant
}

const (
	// DefaultBaseURL is the URL that is used to make requests to the FireHydrant API
	DefaultBaseURL = "https://api.firehydrant.io/v1/"
)

var _ Client = &APIClient{}

// Client is the client that makes requests to FireHydrant
type Client interface {
	Ping(ctx context.Context) (*PingResponse, error)

	Runbooks() RunbooksClient
	RunbookActions() RunbookActionsClient
	ServiceDependencies() ServiceDependenciesClient
	Severities() SeveritiesClient
	TaskLists() TaskListsClient
	SlackChannels() SlackChannelsClient
	StatusUpdateTemplates() StatusUpdateTemplates

	// Users
	GetUsers(ctx context.Context, params GetUserParams) (*UserResponse, error)

	// Signals
	IngestURL() IngestURLClient
	Transposers() TransposersClient
}

type transportWithUserAgent struct {
	userAgent string
}

func (t *transportWithUserAgent) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return http.DefaultTransport.RoundTrip(req)
}

// OptFunc is a function that sets a setting on a client
type OptFunc func(c *APIClient) error

// WithBaseURL modifies the base URL for all requests
func WithBaseURL(baseURL string) OptFunc {
	return func(c *APIClient) error {
		c.baseURL = baseURL
		return nil
	}
}

func WithUserAgentSuffix(suffix string) OptFunc {
	return func(c *APIClient) error {
		c.userAgentSuffix = suffix
		return nil
	}
}

// NewRestClient initializes a new API client for FireHydrant
func NewRestClient(token string, opts ...OptFunc) (*APIClient, error) {
	firehydrantBaseURL := os.Getenv("FIREHYDRANT_BASE_URL")
	if firehydrantBaseURL == "" {
		firehydrantBaseURL = DefaultBaseURL
	}

	c := &APIClient{
		baseURL: firehydrantBaseURL,
		token:   token,
	}

	for _, f := range opts {
		if err := f(c); err != nil {
			return nil, err
		}
	}

	//init speakeasy client also
	httpClient := &http.Client{Transport: &transportWithUserAgent{
		userAgent: fmt.Sprintf("%s (%s)/%s", UserAgentPrefix, GetBuildInfo().String(), c.userAgentSuffix)},
	}

	// speakeasy sdk will only work with v1 of the api and adds this to each path automatically.  The server URL then assumes no path information
	// Thus, we need to strip any trailing 'v1/' from the base URL provided to configure the old client.
	//
	// This reads c.baseURL rather than the environment value it started as, so that
	// WithBaseURL reaches the SDK client too. Reading the original value meant the
	// SDK ignored the option and kept talking to whatever the environment pointed
	// at, which is the default production API when the environment says nothing.
	firehydrantServerURL := strings.TrimSuffix(c.baseURL, "v1/")

	c.Sdk = fhsdk.New(
		fhsdk.WithClient(httpClient),
		fhsdk.WithServerURL(firehydrantServerURL),
		fhsdk.WithSecurity(components.Security{
			APIKey: token,
		}),
	)

	return c, nil
}

func (c *APIClient) client() *sling.Sling {
	bi := GetBuildInfo()

	return sling.New().Base(c.baseURL).
		Set(
			"User-Agent",
			fmt.Sprintf(
				"%s (%s)/%s",
				UserAgentPrefix,
				bi.String(),
				c.userAgentSuffix,
			),
		).
		Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
}

// Ping hits and verifies the HTTP of FireHydrant
func (c *APIClient) Ping(ctx context.Context) (*PingResponse, error) {
	pingResponse := &PingResponse{}
	apiError := &APIError{}
	response, err := c.client().Get("ping").Receive(pingResponse, apiError)
	if err != nil {
		return nil, errors.Wrap(err, "could not ping")
	}

	err = checkResponseStatusCode(response, apiError)
	if err != nil {
		return nil, err
	}

	return pingResponse, nil
}

// Runbooks returns a RunbooksClient interface for interacting with runbooks in FireHydrant
func (c *APIClient) Runbooks() RunbooksClient {
	return &RESTRunbooksClient{client: c}
}

// RunbookActions returns a RunbookActionsClient interface for interacting with runbook actions in FireHydrant
func (c *APIClient) RunbookActions() RunbookActionsClient {
	return &RESTRunbookActionsClient{client: c}
}

// ServiceDependencies returns a ServiceDependenciesClient interface for interacting with service dependencies in FireHydrant
func (c *APIClient) ServiceDependencies() ServiceDependenciesClient {
	return &RESTServiceDependenciesClient{client: c}
}

// Severities returns a SeveritiesClient interface for interacting with severities in FireHydrant
func (c *APIClient) Severities() SeveritiesClient {
	return &RESTSeveritiesClient{client: c}
}

// TaskLists returns a TaskListsClient interface for interacting with task lists in FireHydrant
func (c *APIClient) TaskLists() TaskListsClient {
	return &RESTTaskListsClient{client: c}
}

// SlackChannels returns a SlackChannelsClient interface for interacting with slack channels in FireHydrant
func (c *APIClient) SlackChannels() SlackChannelsClient {
	return &RESTSlackChannelsClient{client: c}
}

// IngestURL returns a IngestURLClient interface for retrieving ingest URLs in FireHydrant
func (c *APIClient) IngestURL() IngestURLClient {
	return &RESTIngestURLClient{client: c}
}

// Transposers returns a TransposersClient interface for retrieving transposers from FireHydrant
func (c *APIClient) Transposers() TransposersClient {
	return &RESTTransposersClient{client: c}
}

func (c *APIClient) StatusUpdateTemplates() StatusUpdateTemplates {
	return &RESTStatusUpdateTemplateClient{client: c}
}

// GetUsers gets matching users in FireHydrant
func (c *APIClient) GetUsers(ctx context.Context, params GetUserParams) (*UserResponse, error) {
	userResponse := &UserResponse{}
	apiError := &APIError{}
	response, err := c.client().Get("users").QueryStruct(params).Receive(userResponse, apiError)
	if err != nil {
		return nil, errors.Wrap(err, "could not get users")
	}

	err = checkResponseStatusCode(response, apiError)
	if err != nil {
		return nil, err
	}

	return userResponse, nil
}
