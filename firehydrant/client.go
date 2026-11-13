package firehydrant

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	fhsdk "github.com/firehydrant/firehydrant-go-sdk"
	"github.com/firehydrant/firehydrant-go-sdk/models/components"
	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/dghubble/sling"
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
	maxRetries      int

	rateLimitedClient *rateLimitedHTTPClient

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

func WithMaxRetries(maxRetries int) OptFunc {
	return func(c *APIClient) error {
		c.maxRetries = maxRetries
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

	client := retryablehttp.NewClient()
	client.RetryMax = c.maxRetries
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client.HTTPClient.Transport = &transportWithUserAgent{
		userAgent: fmt.Sprintf("%s (%s)/%s", UserAgentPrefix, GetBuildInfo().String(), c.userAgentSuffix),
	}
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil {
			if resp.StatusCode != http.StatusInternalServerError {
				return false, nil
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	c.rateLimitedClient = newRateLimitedHTTPClient(client)

	// speakeasy sdk will only work with v1 of the api and adds this to each path automatically.  The server URL then assumes no path information
	// Thus, we need to strip any trailing 'v1/' from the base URL provided to configure the old client.
	firehydrantServerURL := strings.TrimSuffix(firehydrantBaseURL, "v1/")

	c.Sdk = fhsdk.New(
		fhsdk.WithClient(c.rateLimitedClient),
		fhsdk.WithServerURL(firehydrantServerURL),
		fhsdk.WithSecurity(components.Security{
			APIKey: token,
		}),
	)

	return c, nil
}

func (c *APIClient) client() *sling.Sling {
	return sling.New().
		Doer(c.rateLimitedClient).
		Base(c.baseURL).
		Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
}

// Ping hits and verifies the HTTP of FireHydrant
func (c *APIClient) Ping(ctx context.Context) (*PingResponse, error) {
	pingResponse := &PingResponse{}

	if err := c.LegacyGet(ctx, "ping", nil, pingResponse); err != nil {
		return nil, fmt.Errorf("could not ping: %w", err)
	}

	return pingResponse, nil
}

func (c *APIClient) LegacyGet(ctx context.Context, path string, queryParams interface{}, success interface{}) error {
	s := c.client().Get(path)
	if queryParams != nil {
		s = s.QueryStruct(queryParams)
	}
	return c.legacyDo(ctx, s, success)
}

func (c *APIClient) LegacyPost(ctx context.Context, path string, params interface{}, success interface{}) error {
	s := c.client().Post(path)
	if params != nil {
		s = s.BodyJSON(params)
	}
	return c.legacyDo(ctx, s, success)
}

func (c *APIClient) LegacyPut(ctx context.Context, path string, params interface{}, success interface{}) error {
	s := c.client().Put(path)
	if params != nil {
		s = s.BodyJSON(params)
	}
	return c.legacyDo(ctx, s, success)
}

func (c *APIClient) LegacyPatch(ctx context.Context, path string, params interface{}, success interface{}) error {
	s := c.client().Patch(path)
	if params != nil {
		s = s.BodyJSON(params)
	}
	return c.legacyDo(ctx, s, success)
}

// LegacyDelete issues a DELETE to path.
func (c *APIClient) LegacyDelete(ctx context.Context, path string) error {
	return c.legacyDo(ctx, c.client().Delete(path), nil)
}

func (c *APIClient) legacyDo(ctx context.Context, s *sling.Sling, success interface{}) error {
	req, err := s.Request()
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	req = req.WithContext(ctx)

	apiError := &APIError{}
	response, err := s.Do(req, success, apiError)
	if err != nil {
		return err
	}

	return checkResponseStatusCode(response, apiError)
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

	if err := c.LegacyGet(ctx, "users", params, userResponse); err != nil {
		return nil, fmt.Errorf("could not get users: %w", err)
	}

	return userResponse, nil
}
