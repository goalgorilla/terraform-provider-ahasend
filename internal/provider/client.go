// Package provider implements the AhaSend Terraform Plugin Framework provider
// and its managed resources (domains, API keys, webhooks, SMTP credentials,
// and Platform Partner sub accounts).
package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/AhaSend/ahasend-go/api"
	"github.com/google/uuid"
)

// ahasendClient wraps the official SDK client and the configured account ID.
type ahasendClient struct {
	api       *api.APIClient
	accountID uuid.UUID
}

// newAhaSendClient builds an SDK client with retries and rate limiting enabled.
// accountID must be a valid UUID. endpoint may be empty to keep the SDK default.
func newAhaSendClient(apiKey, accountID, endpoint string) (*ahasendClient, error) {
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account_id: %w", err)
	}

	opts := []api.ClientOption{
		api.WithAPIKey(apiKey),
		api.WithRetryConfig(api.DefaultRetryConfig()),
		api.WithRateLimit(true),
	}

	client := api.NewAPIClient(opts...)

	if endpoint != "" {
		if err := configureEndpoint(client, endpoint); err != nil {
			return nil, err
		}
	}

	return &ahasendClient{
		api:       client,
		accountID: parsedAccountID,
	}, nil
}

// configureEndpoint overrides the SDK default host using a full base URL
// such as https://api.ahasend.com (path prefixes are preserved on Servers[0].URL).
func configureEndpoint(client *api.APIClient, endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("endpoint must include scheme and host, got %q", endpoint)
	}

	base := strings.TrimRight(endpoint, "/")
	cfg := client.GetConfig()
	cfg.Scheme = parsed.Scheme
	cfg.Host = parsed.Host
	cfg.Servers = api.ServerConfigurations{
		{
			URL:         base,
			Description: "Configured endpoint",
		},
	}
	return nil
}

// isNotFound reports whether err is an AhaSend API 404 response.
func isNotFound(err error) bool {
	var apiErr *api.APIError
	if err == nil {
		return false
	}
	if asAPIError(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// asAPIError assigns *target when err is an *api.APIError.
func asAPIError(err error, target **api.APIError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*api.APIError); ok {
		*target = e
		return true
	}
	return false
}

// formatAPIError returns a user-facing diagnostic string for API and other errors.
func formatAPIError(err error) string {
	var apiErr *api.APIError
	if asAPIError(err, &apiErr) {
		msg := apiErr.Message
		if msg == "" {
			msg = http.StatusText(apiErr.StatusCode)
			if msg == "" {
				msg = "Unknown HTTP status"
			}
		}
		return fmt.Sprintf("AhaSend API error (HTTP %d): %s", apiErr.StatusCode, msg)
	}
	return err.Error()
}

// stringPtr returns a pointer to v for optional SDK request fields.
func stringPtr(v string) *string {
	return &v
}

// intPtr returns a pointer to v for optional SDK request fields.
func intPtr(v int) *int {
	return &v
}

// boolPtr returns a pointer to v for optional SDK request fields.
func boolPtr(v bool) *bool {
	return &v
}
