package firehydrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pingResponseJSON = `{"response":"pong","actor":{"id":"2af3339f-9d81-434b-a208-427d6d85c124","name":"Bobby Tables","email":"bobby+dalmatians@firehydrant.io","type":"firehydrant_user"}}`
)

type RequestTest func(req *http.Request)

func AssertRequestJSONBody(t *testing.T, src interface{}) RequestTest {
	return func(req *http.Request) {
		t.Run("AssertRequestJSONBody", func(t *testing.T) {
			body := io.NopCloser(req.Body)

			buf := new(bytes.Buffer)
			require.NoError(t, json.NewEncoder(buf).Encode(src))

			// Read the body out so we can compare to what we received
			b, err := io.ReadAll(body)
			require.NoError(t, err)

			assert.Equal(t, buf.Bytes(), b)
		})
	}
}

func AssertRequestMethod(t *testing.T, method string) RequestTest {
	return func(req *http.Request) {
		t.Run("AssertRequestMethod", func(t *testing.T) {
			assert.Equal(t, method, req.Method)
		})
	}
}

func setupClient(requestPath string, mockedResponse interface{}, requestTests ...RequestTest) (*APIClient, func(), error) {
	// We only handle the request path passed in the setup, this ensures that we serve
	// a 404 on any other request, failing the client in a more predictable and easier to
	// debug way
	mux := http.NewServeMux()
	mux.HandleFunc(requestPath, func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		for _, test := range requestTests {
			test(req)
		}

		if err := faker.FakeData(mockedResponse); err != nil {
			panic(fmt.Errorf("could not fake a response: %w", err))
		}

		if err := json.NewEncoder(w).Encode(mockedResponse); err != nil {
			panic(fmt.Errorf("could not encode JSON: %w", err))
		}
	})

	ts := httptest.NewServer(mux)

	c, err := NewRestClient("fake-token", WithBaseURL(ts.URL))
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate rest client: %w", err)
	}

	teardown := func() {
		ts.Close()
	}

	return c, teardown, nil
}

func TestClientInitialization(t *testing.T) {
	var requestPathRcvd, token string

	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestPathRcvd = req.URL.Path
		token = req.Header.Get("Authorization")

		w.Write([]byte(pingResponseJSON))
	})
	ts := httptest.NewServer(h)

	defer ts.Close()

	testToken := "testing-123"
	c, err := NewRestClient(testToken, WithBaseURL(ts.URL))

	if err != nil {
		t.Fatalf("Received error initializing API client: %s", err.Error())
		return
	}

	res, err := c.Ping(context.TODO())
	if err != nil {
		t.Fatalf("Received error hitting ping endpoint: %s", err.Error())
	}

	actorID := res.Actor.ID
	actorEmail := res.Actor.Email

	if expected := "/ping"; expected != requestPathRcvd {
		t.Fatalf("Expected %s, Got: %s for request path", expected, requestPathRcvd)
	}

	if expected := "Bearer " + testToken; expected != token {
		t.Fatalf("Expected %s, Got: %s for bearer token", expected, token)
	}

	if expected := "2af3339f-9d81-434b-a208-427d6d85c124"; expected != actorID {
		t.Fatalf("Expected %s, Got: %s for actor ID", expected, actorID)
	}

	if expected := "bobby+dalmatians@firehydrant.io"; expected != actorEmail {
		t.Fatalf("Expected %s, Got: %s for actor email", expected, actorEmail)
	}
}

func TestClient_RateLimiting(t *testing.T) {
	expectedCallCount := 3
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		statusCode := http.StatusTooManyRequests
		if callCount++; callCount == expectedCallCount {
			statusCode = http.StatusOK
		}

		w.WriteHeader(statusCode)
	}))
	defer ts.Close()

	c, err := NewRestClient("123", WithBaseURL(ts.URL), WithMaxRetries(expectedCallCount))
	assert.NoError(t, err)

	c.httpClient.client.RetryWaitMin = time.Millisecond
	c.httpClient.client.RetryWaitMax = time.Millisecond

	_, err = c.Ping(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, expectedCallCount, callCount)
}

func TestClient_RateLimitingConnectionError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close()

	c, err := NewRestClient("123", WithBaseURL(ts.URL))
	assert.NoError(t, err)

	c.httpClient.client.RetryWaitMin = time.Millisecond
	c.httpClient.client.RetryWaitMax = time.Millisecond

	_, err = c.Ping(t.Context())
	assert.Error(t, err)
}
