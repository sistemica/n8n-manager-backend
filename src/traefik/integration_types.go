package traefik

// EchoResponse represents the structure returned by our echo server
type EchoResponse struct {
	Path    string              `json:"path"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    string              `json:"body"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status"`
}
