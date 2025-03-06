package traefik

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathParamsToHeaderMw(t *testing.T) {
	tests := []struct {
		name       string
		pathParams map[string]string
		expected   map[string]string
	}{
		{
			name: "single parameter",
			pathParams: map[string]string{
				"UserID": "userId",
			},
			expected: map[string]string{
				"X-UserID": "{{ .Route.userId }}",
			},
		},
		{
			name: "multiple parameters",
			pathParams: map[string]string{
				"UserID":    "userId",
				"ProjectID": "projectId",
			},
			expected: map[string]string{
				"X-UserID":    "{{ .Route.userId }}",
				"X-ProjectID": "{{ .Route.projectId }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := PathParamsToHeaderMw(tt.pathParams)
			assert.Equal(t, tt.expected, mw.Headers.CustomRequestHeaders)
		})
	}
}

func TestQueryParamsToHeaderMw(t *testing.T) {
	tests := []struct {
		name        string
		queryParams []string
		expected    map[string]string
	}{
		{
			name:        "single parameter",
			queryParams: []string{"version"},
			expected: map[string]string{
				"X-version": "{{ .Query.version }}",
			},
		},
		{
			name:        "multiple parameters",
			queryParams: []string{"version", "format", "lang"},
			expected: map[string]string{
				"X-version": "{{ .Query.version }}",
				"X-format":  "{{ .Query.format }}",
				"X-lang":    "{{ .Query.lang }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := QueryParamsToHeaderMw(tt.queryParams)
			assert.Equal(t, tt.expected, mw.Headers.CustomRequestHeaders)
		})
	}
}

func TestBasicAuthRateLimitMw(t *testing.T) {
	mw := BasicAuthMw("testuser", "testpass")

	t.Run("basic auth configuration", func(t *testing.T) {
		assert.NotNil(t, mw.BasicAuth)
		assert.Len(t, mw.BasicAuth.Users, 1)
		assert.Equal(t, "Protected API", mw.BasicAuth.Realm)
	})
}

func TestRateLimitMw(t *testing.T) {
	mw := RateLimitMw(100, 50)

	t.Run("rate limit configuration", func(t *testing.T) {
		assert.NotNil(t, mw.RateLimit)
		assert.Equal(t, 100, mw.RateLimit.Average)
		assert.Equal(t, 50, mw.RateLimit.Burst)
		assert.Equal(t, "1m", mw.RateLimit.Period)
	})
}

func TestAPIKeyMw(t *testing.T) {
	tests := []struct {
		name       string
		headerName string
		apiKey     string
	}{
		{
			name:       "standard header",
			headerName: "X-API-Key",
			apiKey:     "test-key",
		},
		{
			name:       "custom header",
			headerName: "X-Custom-Token",
			apiKey:     "custom-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := APIKeyMw(tt.headerName, tt.apiKey)
			assert.NotNil(t, mw.Headers)
			assert.Equal(t, tt.apiKey, mw.Headers.CustomRequestHeaders[tt.headerName])
		})
	}
}
