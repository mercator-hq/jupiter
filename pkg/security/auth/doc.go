/*
Package auth provides API key authentication and validation for Mercator Jupiter.

This package implements HTTP middleware for validating API keys from various sources
(headers, query parameters) and provides a flexible validation system with support
for rate limiting integration.

# Basic Usage

Create an API key validator and middleware:

	validator := auth.NewAPIKeyValidator([]*auth.APIKeyInfo{
		{
			Key:       "sk-test-1234567890abcdef",
			UserID:    "user-123",
			TeamID:    "team-engineering",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	})

	sources := []auth.APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
		{Type: "header", Name: "X-API-Key", Scheme: ""},
		{Type: "query", Name: "api_key", Scheme: ""},
	}

	middleware := auth.NewAPIKeyMiddleware(validator, sources)

	// Wrap your handler
	http.Handle("/api/", middleware.Handle(yourHandler))

# Extracting API Key Info

Inside your HTTP handler, retrieve the authenticated user's information:

	func handler(w http.ResponseWriter, r *http.Request) {
		keyInfo, ok := auth.GetAPIKeyInfo(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Printf("Request from user %s (team: %s)\n", keyInfo.UserID, keyInfo.TeamID)
	}

# API Key Sources

The middleware supports multiple sources for API keys:

 1. Authorization header with Bearer scheme:
    Authorization: Bearer sk-test-1234567890abcdef

 2. Custom header:
    X-API-Key: sk-test-1234567890abcdef

 3. Query parameter:
    ?api_key=sk-test-1234567890abcdef

The middleware tries sources in order and uses the first valid key found.

# Security Considerations

- API key values are never logged (only user/team IDs)
- Use HTTPS in production to prevent key interception
- Rotate API keys regularly (90 days recommended)
- Generate cryptographically random keys (min 32 bytes)
- Integrate with rate limiting to prevent brute force attacks
- Monitor authentication failures for suspicious activity

# Configuration Example

	security:
	  authentication:
	    enabled: true
	    sources:
	      - type: "header"
	        name: "Authorization"
	        scheme: "Bearer"
	      - type: "header"
	        name: "X-API-Key"
	        scheme: ""
	    keys:
	      - key: "sk-test-1234567890abcdef"
	        user_id: "user-123"
	        team_id: "team-engineering"
	        enabled: true
	        rate_limit: "1000/hour"
*/
package auth
