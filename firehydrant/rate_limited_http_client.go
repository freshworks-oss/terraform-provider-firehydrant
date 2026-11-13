package firehydrant

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

var expensiveEndpoints = map[string][]string{
	"/v1/services": {http.MethodGet},
	"/v1/teams":    {http.MethodGet},
}

const expensiveEndpointsRateLimitBudget = .9

type rateLimitedHTTPClient struct {
	client           *retryablehttp.Client
	commonLimiter    *rate.Limiter
	expensiveLimiter *rate.Limiter
}

func newRateLimitedHTTPClient(client *retryablehttp.Client) *rateLimitedHTTPClient {
	return &rateLimitedHTTPClient{
		client:           client,
		commonLimiter:    rate.NewLimiter(rate.Every(time.Minute/60000), 1),
		expensiveLimiter: rate.NewLimiter(rate.Every(5*time.Second/10)*expensiveEndpointsRateLimitBudget, 1),
	}
}

func (hc *rateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if slices.Contains(expensiveEndpoints[req.URL.Path], req.Method) {
		if err := hc.expensiveLimiter.Wait(req.Context()); err != nil {
			return nil, fmt.Errorf("error waiting for client rate limiter: %w", err)
		}
	} else if err := hc.commonLimiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("error waiting for client rate limiter: %w", err)
	}

	r, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}

	return hc.client.Do(r)
}
