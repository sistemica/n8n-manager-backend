package traefik

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// parseURL splits a URL into host and port
func parseURL(url string) (host string, port string) {
	// Remove protocol if present
	if strings.HasPrefix(url, "http://") {
		url = url[7:]
	} else if strings.HasPrefix(url, "https://") {
		url = url[8:]
	}

	// Split host and port
	parts := strings.Split(url, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "80"
}

// buildAuthHeader creates a base64 encoded basic auth header
func buildAuthHeader(username, password string) string {
	auth := fmt.Sprintf("%s:%s", username, password)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
