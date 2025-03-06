// builder_test.go
package traefik

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceNamer(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "simple path",
			parts:    []string{"example.com", "/api/users", "router"},
			expected: "example-com-api-users-router",
		},
		{
			name:     "path with parameters",
			parts:    []string{"api.example.com", "/api/users/{userId}", "middleware"},
			expected: "api-example-com-api-users-userid-middleware",
		},
		{
			name:     "multiple dashes",
			parts:    []string{"test.example.com", "/api//users", "service"},
			expected: "test-example-com-api-users-service",
		},
		{
			name:     "special characters",
			parts:    []string{"test:123", "/api_{version}", "router"},
			expected: "test-123-api-version-router",
		},
	}

	namer := NewResourceNamer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := namer.generateName(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildServiceURL(t *testing.T) {
	tests := []struct {
		name     string
		service  ServiceDefinition
		expected string
	}{
		{
			name: "no scheme specified",
			service: ServiceDefinition{
				Host: "localhost",
				Port: 8080,
			},
			expected: "http://localhost:8080",
		},
		{
			name: "http scheme",
			service: ServiceDefinition{
				Host:   "localhost",
				Port:   8080,
				Scheme: "http",
			},
			expected: "http://localhost:8080",
		},
		{
			name: "https scheme",
			service: ServiceDefinition{
				Host:   "localhost",
				Port:   8443,
				Scheme: "https",
			},
			expected: "https://localhost:8443",
		},
		{
			name: "custom domain without scheme",
			service: ServiceDefinition{
				Host: "api.example.com",
				Port: 9000,
			},
			expected: "http://api.example.com:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := buildServiceURL(tt.service)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestBuilder(t *testing.T) {
	tests := []struct {
		name  string
		route RouteDefinition
		check func(t *testing.T, config *DynamicConfig)
	}{
		{
			name: "basic route without middleware",
			route: RouteDefinition{
				Host: "example.com",
				Path: "/api",
				Service: ServiceDefinition{
					Host: "backend",
					Port: 8080,
				},
				EntryPoints: []string{"web"},
			},
			check: func(t *testing.T, config *DynamicConfig) {
				// Check router
				router, exists := config.HTTP.Routers["example-com-api-router"]
				require.True(t, exists)
				assert.Equal(t, "Host(`example.com`) && Path(`/api`)", router.Rule)
				assert.Equal(t, []string{"web"}, router.EntryPoints)
				assert.Empty(t, router.Middlewares)

				// Check service
				service, exists := config.HTTP.Services["example-com-api-service"]
				require.True(t, exists)
				assert.Equal(t, "http://backend:8080", service.LoadBalancer.Servers[0].URL)
			},
		},
		{
			name: "route with path parameters",
			route: RouteDefinition{
				Host: "api.example.com",
				Path: "/users/{userId}",
				PathParams: map[string]string{
					"UserID": "userId",
				},
				Service: ServiceDefinition{
					Host:   "users-service",
					Port:   8081,
					Scheme: "http",
				},
			},
			check: func(t *testing.T, config *DynamicConfig) {
				router, exists := config.HTTP.Routers["api-example-com-users-userid-router"]
				require.True(t, exists)
				assert.Contains(t, router.Middlewares, "api-example-com-users-userid-path-params-middleware")

				mw, exists := config.HTTP.Middlewares["api-example-com-users-userid-path-params-middleware"]
				require.True(t, exists)
				assert.Equal(t, "{{ .Route.userId }}", mw.Headers.CustomRequestHeaders["X-UserID"])

				service, exists := config.HTTP.Services["api-example-com-users-userid-service"]
				require.True(t, exists)
				assert.Equal(t, "http://users-service:8081", service.LoadBalancer.Servers[0].URL)
			},
		},
		{
			name: "route with basic auth",
			route: RouteDefinition{
				Host: "secure.example.com",
				Path: "/admin",
				Service: ServiceDefinition{
					Host:   "admin-service",
					Port:   8443,
					Scheme: "https",
				},
				Authentication: &AuthConfig{
					Type:     "basic",
					Username: "admin",
					Password: "secret",
				},
			},
			check: func(t *testing.T, config *DynamicConfig) {
				router, exists := config.HTTP.Routers["secure-example-com-admin-router"]
				require.True(t, exists)
				assert.Contains(t, router.Middlewares, "secure-example-com-admin-basic-auth-middleware")

				mw, exists := config.HTTP.Middlewares["secure-example-com-admin-basic-auth-middleware"]
				require.True(t, exists)
				assert.NotNil(t, mw.BasicAuth)

				service, exists := config.HTTP.Services["secure-example-com-admin-service"]
				require.True(t, exists)
				assert.Equal(t, "https://admin-service:8443", service.LoadBalancer.Servers[0].URL)
			},
		},
		{
			name: "route with query parameters",
			route: RouteDefinition{
				Host:        "api.example.com",
				Path:        "/search",
				QueryParams: []string{"q", "lang"},
				Service: ServiceDefinition{
					Host: "search-service",
					Port: 8080,
				},
			},
			check: func(t *testing.T, config *DynamicConfig) {
				router, exists := config.HTTP.Routers["api-example-com-search-router"]
				require.True(t, exists)
				assert.Contains(t, router.Middlewares, "api-example-com-search-query-params-middleware")

				mw, exists := config.HTTP.Middlewares["api-example-com-search-query-params-middleware"]
				require.True(t, exists)
				assert.Equal(t, "{{ .Query.q }}", mw.Headers.CustomRequestHeaders["X-q"])
				assert.Equal(t, "{{ .Query.lang }}", mw.Headers.CustomRequestHeaders["X-lang"])

				service, exists := config.HTTP.Services["api-example-com-search-service"]
				require.True(t, exists)
				assert.Equal(t, "http://search-service:8080", service.LoadBalancer.Servers[0].URL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder()
			config := builder.Build([]RouteDefinition{tt.route})
			tt.check(t, config)
		})
	}
}
