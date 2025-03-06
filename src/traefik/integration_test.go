package traefik

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check required environment variables
	portStr := os.Getenv(TestPort)
	if portStr == "" {
		t.Fatalf("Required environment variable not set: %s", TestPort)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("Invalid port number %s: %v", portStr, err)
	}

	// Define test routes
	routes := []RouteDefinition{
		{
			Host:        "health.example.com",
			Path:        "/health",
			EntryPoints: []string{"web"},
			Service: ServiceDefinition{
				Host: "host.docker.internal",
				Port: port,
				// No scheme - let Traefik handle it
			},
		},
		{
			Host: "example.com",
			Path: "/api/v1/users/{userId}",
			PathParams: map[string]string{
				"UserID": "userId",
			},
			QueryParams: []string{"version"},
			EntryPoints: []string{"web"},
			Service: ServiceDefinition{
				Host:   "host.docker.internal",
				Port:   port,
				Scheme: "http", // Explicit HTTP
			},
			Authentication: &AuthConfig{
				Type:     "basic",
				Username: "admin",
				Password: "secret",
			},
		},
		{
			Host: "api.example.com",
			Path: "/products/{category}/{id}",
			PathParams: map[string]string{
				"Category": "category",
				"ID":       "id",
			},
			QueryParams: []string{"currency", "lang"},
			EntryPoints: []string{"web"},
			Service: ServiceDefinition{
				Host:   "host.docker.internal",
				Port:   port,
				Scheme: "http", // Explicit http
			},
		},
	}

	// Start test server
	server, serverURL := startTestServer(t, routes)
	defer server.Shutdown(context.Background())

	// Wait for server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()
	err = waitForHTTP(ctx, serverURL+"/health")
	require.NoError(t, err, "Server health check failed")

	// Wait for Traefik to be ready
	ctx, cancel = context.WithTimeout(context.Background(), traefikTimeout)
	defer cancel()
	err = waitForTraefik(ctx)
	if err != nil {
		t.Logf("Traefik not ready: %v", err)
		t.SkipNow()
	}

	// Wait for Traefik health check route to be ready
	ctx, cancel = context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()
	healthURL := "http://localhost/health"
	req, _ := http.NewRequest("GET", healthURL, nil)
	req.Host = "health.example.com"
	err = waitForHealthEndpoint(ctx, req)
	if err != nil {
		t.Logf("Traefik health check route not ready: %v", err)
		t.SkipNow()
	}

	// Test cases
	t.Run("protected route with auth", func(t *testing.T) {
		// Test without auth
		req, _ := http.NewRequest("GET", "http://localhost/api/v1/users/123?version=2", nil)
		req.Host = "example.com"
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to make request")
		if resp != nil {
			defer resp.Body.Close()
		}
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// Test with auth
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("admin:secret")))
		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to make authenticated request")
		if resp == nil {
			t.Fatal("No response received")
		}
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		var echo EchoResponse
		err = json.Unmarshal(body, &echo)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "/api/v1/users/123", echo.Path)
		assert.Contains(t, echo.Headers, "X-Userid")
		assert.Equal(t, []string{"123"}, echo.Headers["X-Userid"])
		assert.Contains(t, echo.Headers, "X-Version")
		assert.Equal(t, []string{"2"}, echo.Headers["X-Version"])
	})

	t.Run("product endpoint with multiple parameters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost/products/electronics/12345?currency=EUR&lang=de", nil)
		req.Host = "api.example.com"
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Failed to make request")
		if resp == nil {
			t.Fatal("No response received")
		}
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Unexpected status code")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Failed to read response body")

		var echo EchoResponse
		err = json.Unmarshal(body, &echo)
		require.NoError(t, err, "Failed to decode response")

		assert.Equal(t, "/products/electronics/12345", echo.Path)
		assert.Contains(t, echo.Headers, "X-Category")
		assert.Equal(t, []string{"electronics"}, echo.Headers["X-Category"])
		assert.Contains(t, echo.Headers, "X-Id")
		assert.Equal(t, []string{"12345"}, echo.Headers["X-Id"])
		assert.Contains(t, echo.Headers, "X-Currency")
		assert.Equal(t, []string{"EUR"}, echo.Headers["X-Currency"])
		assert.Contains(t, echo.Headers, "X-Lang")
		assert.Equal(t, []string{"de"}, echo.Headers["X-Lang"])
	})
	time.Sleep(500 * time.Second)
}
