package firehydrant

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func blockedLimiter() *rate.Limiter { return rate.NewLimiter(rate.Every(time.Hour), 0) }

func openLimiter() *rate.Limiter { return rate.NewLimiter(rate.Inf, 0) }

func TestRateLimitedHTTPClient_LimiterSelection(t *testing.T) {
	tests := []struct {
		Name            string
		Method          string
		Path            string
		ExpectExpensive bool
	}{
		{
			Name:            "GET services",
			Method:          http.MethodGet,
			Path:            "/v1/services",
			ExpectExpensive: true,
		},
		{
			Name:            "GET teams",
			Method:          http.MethodGet,
			Path:            "/v1/teams",
			ExpectExpensive: true,
		},
		{
			Name:            "GET teams with query string",
			Method:          http.MethodGet,
			Path:            "/v1/teams?query=foo",
			ExpectExpensive: true,
		},
		{
			Name:            "POST teams",
			Method:          http.MethodPost,
			Path:            "/v1/teams",
			ExpectExpensive: false,
		},
		{
			Name:            "DELETE services",
			Method:          http.MethodDelete,
			Path:            "/v1/services",
			ExpectExpensive: false,
		},
		{
			Name:            "GET single team",
			Method:          http.MethodGet,
			Path:            "/v1/teams/123",
			ExpectExpensive: false,
		},
		{
			Name:            "GET other endpoint",
			Method:          http.MethodGet,
			Path:            "/v1/flux-capacitors",
			ExpectExpensive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var hits atomic.Int32
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusTeapot)
			}))
			defer ts.Close()

			client := newRateLimitedHTTPClient(retryablehttp.NewClient())
			if tt.ExpectExpensive {
				client.commonLimiter = blockedLimiter()
				client.expensiveLimiter = openLimiter()
			} else {
				client.commonLimiter = openLimiter()
				client.expensiveLimiter = blockedLimiter()
			}

			req, err := http.NewRequestWithContext(context.Background(), tt.Method, fmt.Sprintf("%s%s", ts.URL, tt.Path), nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}

			require.NoError(t, err)
			assert.EqualValues(t, 1, hits.Load())
		})
	}
}

func TestRateLimitedHTTPClient_ThrottlesRequests(t *testing.T) {
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTeapot)
	}))
	defer ts.Close()

	const interval = 50 * time.Millisecond
	client := newRateLimitedHTTPClient(retryablehttp.NewClient())
	client.commonLimiter = rate.NewLimiter(rate.Every(interval), 1)

	start := time.Now()
	for i := 0; i < 4; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s/v1/flux-capacitors", ts.URL), nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	assert.GreaterOrEqual(t, time.Since(start), 3*interval-10*time.Millisecond)
	assert.EqualValues(t, 4, hits.Load())
}
