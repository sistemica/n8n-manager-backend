package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	healthCheckTimeout  = 30 * time.Second
	healthCheckInterval = time.Second
	traefikTimeout      = 30 * time.Second
)

// waitForHTTP waits for a HTTP endpoint to be available
func waitForHTTP(ctx context.Context, url string) error {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", url)
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

// waitForTraefik checks if Traefik is ready by verifying its headers
func waitForTraefik(ctx context.Context) error {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Traefik")
		case <-ticker.C:
			req, _ := http.NewRequest("GET", "http://localhost", nil)
			req.Host = "non-existent-host.local" // Should get Traefik 404 with headers
			resp, err := http.DefaultClient.Do(req)

			if err != nil {
				// Log error for debugging purposes
				fmt.Printf("Error checking Traefik: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			// Check if Traefik returns the expected 404 response
			if resp.StatusCode == 404 && resp.Header.Get("X-Content-Type-Options") == "nosniff" {
				return nil
			} else {
				// Log response headers for debugging
				fmt.Printf("Unexpected response: %d, Headers: %v\n", resp.StatusCode, resp.Header)
			}
		}
	}
}

// waitForHealthEndpoint waits for the health endpoint through Traefik to be ready
func waitForHealthEndpoint(ctx context.Context, req *http.Request) error {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for health endpoint")
		case <-ticker.C:
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil && resp.StatusCode == http.StatusOK {
					var health HealthResponse
					if json.Unmarshal(body, &health) == nil && health.Status == "ok" {
						return nil
					}
				}
			}
		}
	}
}
