package traefik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

// TestPort is the environment variable name for the test server port
const TestPort = "TRAEFIK_TEST_PORT"

// startTestServer starts a combined server that:
// - Serves Traefik dynamic configuration at /api/config
// - Provides a health check endpoint at /health
// - Acts as an echo server for all other paths to verify Traefik routing
func startTestServer(t *testing.T, routes []RouteDefinition) (*http.Server, string) {
	// Get port from environment
	portStr := os.Getenv(TestPort)
	if portStr == "" {
		t.Fatalf("Environment variable %s not set", TestPort)
	}

	// Convert port to integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("Invalid port number %s: %v", portStr, err)
	}

	// Create router
	mux := http.NewServeMux()

	// Flag to track if config has been logged
	var configLogged bool

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	})

	// Traefik dynamic configuration endpoint
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		// Update service URLs to point to this test server
		for i := range routes {
			routes[i].Service.Port = port
		}

		// Build configuration
		builder := NewBuilder()
		config := builder.Build(routes)

		// Log config only on first request
		if !configLogged {
			configJSON, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				t.Logf("Error marshaling config for debug: %v", err)
			} else {
				t.Logf("Initial Traefik configuration:\n%s", string(configJSON))
				t.Log("Service URLs:")
				for name, svc := range config.HTTP.Services {
					for _, server := range svc.LoadBalancer.Servers {
						t.Logf("  %s -> %s", name, server.URL)
					}
				}
			}
			configLogged = true
		}

		// Serve the config
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

		if err := json.NewEncoder(w).Encode(config); err != nil {
			t.Errorf("Failed to encode config: %v", err)
		}
	})

	// Echo server endpoint (catch-all)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't echo requests to /api/config or /health
		if r.URL.Path == "/api/config" || r.URL.Path == "/health" {
			http.NotFound(w, r)
			return
		}

		// Log incoming request for debugging
		t.Logf("Echo server received request: %s %s", r.Method, r.URL.Path)
		t.Logf("Headers: %v", r.Header)

		// Read and store request body if present
		body := ""
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			body = string(bodyBytes)
		}

		response := EchoResponse{
			Path:    r.URL.Path,
			Headers: r.Header,
			Query:   r.URL.Query(),
			Body:    body,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	})

	// Create and start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", portStr),
		Handler: mux,
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	baseURL := fmt.Sprintf("http://localhost:%s", portStr)
	t.Logf("Test server started at %s", baseURL)
	t.Logf("- Health check: %s/health", baseURL)
	t.Logf("- Config endpoint: %s/api/config", baseURL)
	t.Logf("- Echo server: %s/*", baseURL)

	return server, baseURL
}
