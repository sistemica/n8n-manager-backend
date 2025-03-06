package traefik

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PathParamsToHeaderMw creates a middleware that converts path parameters to headers.
// Example:
//
//	For path "/users/{id}" and pathParams {"UserID": "id"}
//	A request to "/users/123" will add header "X-UserID: 123"
func PathParamsToHeaderMw(pathParams map[string]string) Middleware {
	headers := make(map[string]string)
	for headerName, routeParam := range pathParams {
		headers[fmt.Sprintf("X-%s", headerName)] = fmt.Sprintf("{{ .Route.%s }}", routeParam)
	}

	return Middleware{
		Headers: &Headers{
			CustomRequestHeaders: headers,
		},
	}
}

// QueryParamsToHeaderMw creates a middleware that converts query parameters to headers.
// Example:
//
//	For queryParams ["version"]
//	A request with "?version=v1" will add header "X-version: v1"
func QueryParamsToHeaderMw(queryParams []string) Middleware {
	headers := make(map[string]string)
	for _, param := range queryParams {
		headers[fmt.Sprintf("X-%s", param)] = fmt.Sprintf("{{ .Query.%s }}", param)
	}

	return Middleware{
		Headers: &Headers{
			CustomRequestHeaders: headers,
		},
	}
}

// BasicAuthRateLimitMw creates a middleware combining basic authentication and rate limiting.
// It adds basic auth protection and limits request rates to prevent abuse.
//
//	username: basic auth username
//	password: basic auth password
func BasicAuthMw(username, password string) Middleware {
	authStr := fmt.Sprintf("%s:%s", username, password)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err == nil {
		authStr = fmt.Sprintf("%s:%s", username, string(hashedPassword))
	}

	return Middleware{
		BasicAuth: &BasicAuth{
			Users: []string{authStr},
			Realm: "Protected API",
		},
	}
}

// RateLimitMw creates a middleware for rate limiting
//
//	rateAvg: average requests per minute allowed
//	rateBurst: maximum requests allowed in a burst
func RateLimitMw(average, burst int) Middleware {
	return Middleware{
		RateLimit: &RateLimit{
			Average: average,
			Burst:   burst,
			Period:  "1m",
		},
	}
}

// APIKeyMw creates a middleware that adds API key authentication via headers.
// Example:
//
//	APIKeyMw("X-API-Key", "secret-key")
//	Requires requests to include header "X-API-Key: secret-key"
func APIKeyMw(headerName, apiKey string) Middleware {
	return Middleware{
		Headers: &Headers{
			CustomRequestHeaders: map[string]string{
				headerName: apiKey,
			},
		},
	}
}

// StripPrefixMW removes a part of the incoming request URL.
// TODO
