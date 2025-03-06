// builder.go

package traefik

import (
	"fmt"
	"strings"
)

// ResourceNamer handles the generation of consistent and unique names
// for Traefik resources (routers, services, and middlewares).
type ResourceNamer struct {
	// nameCache stores previously generated names to ensure uniqueness
	nameCache map[string]string
}

// NewResourceNamer creates a new ResourceNamer instance.
func NewResourceNamer() *ResourceNamer {
	return &ResourceNamer{
		nameCache: make(map[string]string),
	}
}

// generateName creates a normalized name from multiple parts.
// It handles special characters, ensures lowercase, and removes duplicated separators.
func (n *ResourceNamer) generateName(parts ...string) string {
	fullName := strings.Join(parts, "-")
	fullName = strings.ToLower(fullName)

	replacer := strings.NewReplacer(
		".", "-",
		"/", "-",
		"_", "-",
		"{", "",
		"}", "",
		":", "-",
	)
	name := replacer.Replace(fullName)

	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	return strings.Trim(name, "-")
}

// getRouterName generates a unique name for a router based on host and path.
func (n *ResourceNamer) getRouterName(rd RouteDefinition) string {
	return n.generateName(rd.Host, rd.Path, "router")
}

// getServiceName generates a unique name for a service based on host and path.
func (n *ResourceNamer) getServiceName(rd RouteDefinition) string {
	return n.generateName(rd.Host, rd.Path, "service")
}

// getMiddlewareName generates a unique name for a middleware based on host, path, and type.
func (n *ResourceNamer) getMiddlewareName(rd RouteDefinition, mwType string) string {
	return n.generateName(rd.Host, rd.Path, mwType, "middleware")
}

// Builder constructs Traefik's dynamic configuration from route definitions.
type Builder struct {
	namer *ResourceNamer
}

// NewBuilder creates a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{
		namer: NewResourceNamer(),
	}
}

// Build generates a complete Traefik dynamic configuration from route definitions.
// It creates all necessary routers, services, and middlewares based on the provided routes.
func (b *Builder) Build(routes []RouteDefinition) *DynamicConfig {
	config := &DynamicConfig{}
	config.HTTP.Routers = make(map[string]Router)
	config.HTTP.Services = make(map[string]Service)
	config.HTTP.Middlewares = make(map[string]Middleware)

	for _, route := range routes {
		b.addRoute(route, config)
	}

	return config
}

// buildServiceURL creates the backend service URL without enforcing a specific scheme
// This allows Traefik to handle the protocol internally
func buildServiceURL(svc ServiceDefinition) string {
	if svc.Scheme == "" {
		// Default to http if no scheme is provided
		svc.Scheme = "http"
	}
	// Use scheme-less URL format to let Traefik handle the protocol
	return fmt.Sprintf("%s://%s:%d", svc.Scheme, svc.Host, svc.Port)
}

// addRoute adds a single route configuration to the dynamic config.
// It creates the router, service, and any necessary middlewares.
func (b *Builder) addRoute(rd RouteDefinition, config *DynamicConfig) {
	routerName := b.namer.getRouterName(rd)
	serviceName := b.namer.getServiceName(rd)
	var middlewares []string

	// Path params middleware
	if len(rd.PathParams) > 0 {
		mwName := b.namer.getMiddlewareName(rd, "path-params")
		config.HTTP.Middlewares[mwName] = PathParamsToHeaderMw(rd.PathParams)
		middlewares = append(middlewares, mwName)
	}

	// Query params middleware
	if len(rd.QueryParams) > 0 {
		mwName := b.namer.getMiddlewareName(rd, "query-params")
		config.HTTP.Middlewares[mwName] = QueryParamsToHeaderMw(rd.QueryParams)
		middlewares = append(middlewares, mwName)
	}

	// Auth middleware
	if rd.Authentication != nil {
		switch rd.Authentication.Type {
		case "basic":
			// Split basic auth and rate limit into separate middlewares
			authMwName := b.namer.getMiddlewareName(rd, "basic-auth")
			rateMwName := b.namer.getMiddlewareName(rd, "rate-limit")

			config.HTTP.Middlewares[authMwName] = BasicAuthMw(
				rd.Authentication.Username,
				rd.Authentication.Password,
			)
			config.HTTP.Middlewares[rateMwName] = RateLimitMw(100, 50)

			middlewares = append(middlewares, authMwName, rateMwName)
		case "apikey":
			mwName := b.namer.getMiddlewareName(rd, "apikey")
			config.HTTP.Middlewares[mwName] = APIKeyMw(
				"X-API-Key",
				rd.Authentication.APIKey,
			)
			middlewares = append(middlewares, mwName)
		}
	}

	// Create router rule combining host and path matching
	hostRule := fmt.Sprintf("Host(`%s`)", rd.Host)
	pathRule := fmt.Sprintf("Path(`%s`)", rd.Path)

	// Add router with combined rules
	config.HTTP.Routers[routerName] = Router{
		EntryPoints: rd.EntryPoints,
		Service:     serviceName,
		Rule:        fmt.Sprintf("%s && %s", hostRule, pathRule),
		Middlewares: middlewares,
	}

	// Add service with protocol-aware URL
	config.HTTP.Services[serviceName] = Service{
		LoadBalancer: &LoadBalancer{
			Servers: []Server{
				{
					URL: buildServiceURL(rd.Service),
				},
			},
		},
	}
}
