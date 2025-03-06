// Package traefik provides functionality to generate Traefik dynamic configurations
// with a focus on converting REST-style routes to Traefik's configuration format.
package traefik

// RouteDefinition defines a complete route configuration including path parameters,
// authentication, and service details. It's used to generate Traefik's dynamic configuration.
type RouteDefinition struct {
	// Host specifies the domain for the route (e.g., "example.com")
	Host string

	// Path defines the URL path pattern including parameters (e.g., "/api/v1/users/{userId}")
	Path string

	// PathParams maps header names to path parameter names
	// Example: {"UserID": "userId"} will create header "X-UserID" from path param "userId"
	PathParams map[string]string

	// QueryParams lists query parameters to convert to headers
	// Example: ["version"] will create header "X-version" from query param "version"
	QueryParams []string

	// EntryPoints lists Traefik entrypoints to use (e.g., ["web", "websecure"])
	EntryPoints []string

	// Service defines the backend service configuration
	Service ServiceDefinition

	// Authentication defines optional auth configuration (basic auth or API key)
	Authentication *AuthConfig
}

// ServiceDefinition contains backend service configuration details
type ServiceDefinition struct {
	// Host is the hostname or IP of the backend service
	Host string

	// Port is the port number the backend service listens on
	Port   int
	Scheme string // http, https, or empty
}

// AuthConfig defines authentication configuration for a route
type AuthConfig struct {
	// Type specifies the authentication type ("basic" or "apikey")
	Type string

	// Username for basic authentication
	Username string

	// Password for basic authentication
	Password string

	// APIKey for API key authentication
	APIKey string
}
