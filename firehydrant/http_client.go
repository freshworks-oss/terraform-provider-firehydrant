package firehydrant

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

type httpClient struct {
	client *retryablehttp.Client
}

func newHttpClient(maxRetries int, userAgent string) *httpClient {
	client := retryablehttp.NewClient()
	client.RetryMax = maxRetries
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client.HTTPClient.Transport = newRateLimitedTransport(userAgent)
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil {
			if resp.StatusCode == http.StatusInternalServerError {
				return false, nil
			}
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	return &httpClient{
		client: client,
	}
}

func (hc *httpClient) Do(req *http.Request) (*http.Response, error) {
	r, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}

	return hc.client.Do(r)
}

var expensiveEndpoints = map[string][]string{
	"/v1/services": {http.MethodGet},
	"/v1/teams":    {http.MethodGet},
}

const expensiveEndpointsRateLimitBudget = .9

type rateLimitedTransport struct {
	userAgent string

	commonLimiter    *rate.Limiter
	expensiveLimiter *rate.Limiter
}

func newRateLimitedTransport(userAgent string) http.RoundTripper {
	return &rateLimitedTransport{
		userAgent:        userAgent,
		commonLimiter:    rate.NewLimiter(rate.Every(time.Minute/60000), 1),
		expensiveLimiter: rate.NewLimiter(rate.Every(5*time.Second/10)*expensiveEndpointsRateLimitBudget, 1),
	}
}

func (r *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", r.userAgent)

	if slices.Contains(expensiveEndpoints[req.URL.Path], req.Method) {
		if err := r.expensiveLimiter.Wait(req.Context()); err != nil {
			return nil, fmt.Errorf("error waiting for client rate limiter: %w", err)
		}
	} else if err := r.commonLimiter.Wait(req.Context()); err != nil {
		return nil, fmt.Errorf("error waiting for client rate limiter: %w", err)
	}

	return http.DefaultTransport.RoundTrip(req)
}
