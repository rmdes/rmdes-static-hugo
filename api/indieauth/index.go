package indieauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// TokenResponse represents the response from the IndieAuth token endpoint
type TokenResponse struct {
	Me          string `json:"me"`
	ClientID    string `json:"client_id"`
	Scope       string `json:"scope"`
	IssuedAt    int64  `json:"issued_at,omitempty"`
	IssuedBy    string `json:"issued_by,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

// ErrorResponse represents an error from token verification
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// VerifyToken verifies an IndieAuth bearer token with the token endpoint
func VerifyToken(token, tokenEndpoint, expectedMe string) (*TokenResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	// Default token endpoint
	if tokenEndpoint == "" {
		tokenEndpoint = "https://tokens.indieauth.com/token"
	}

	// Create request to token endpoint
	req, err := http.NewRequest("GET", tokenEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("token verification failed: %s - %s", errResp.Error, errResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token verification failed with status %d", resp.StatusCode)
	}

	// Parse successful response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Verify the token is for our site
	if expectedMe != "" {
		// Normalize URLs for comparison (with/without trailing slash)
		tokenMe := strings.TrimSuffix(tokenResp.Me, "/")
		expectedNorm := strings.TrimSuffix(expectedMe, "/")

		if tokenMe != expectedNorm {
			return nil, fmt.Errorf("token not issued for this site: got %s, expected %s", tokenResp.Me, expectedMe)
		}
	}

	return &tokenResp, nil
}

// ExtractToken extracts the bearer token from an HTTP request
// Checks Authorization header first, then access_token form value
func ExtractToken(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check form value (for form-encoded requests)
	if err := r.ParseForm(); err == nil {
		if token := r.FormValue("access_token"); token != "" {
			return token
		}
	}

	return ""
}

// HasScope checks if the token response includes a specific scope
func HasScope(tokenResp *TokenResponse, scope string) bool {
	if tokenResp == nil {
		return false
	}

	scopes := strings.Fields(tokenResp.Scope)
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// GetExpectedMe returns the expected "me" URL from environment
func GetExpectedMe() string {
	return os.Getenv("INDIEAUTH_ME")
}

// GetTokenEndpoint returns the token endpoint from environment or default
func GetTokenEndpoint() string {
	endpoint := os.Getenv("INDIEAUTH_TOKEN_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://tokens.indieauth.com/token"
	}
	return endpoint
}
