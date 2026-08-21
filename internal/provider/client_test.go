package provider

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AhaSend/ahasend-go/api"
)

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
		{name: "404", err: &api.APIError{StatusCode: http.StatusNotFound, Message: "missing"}, want: true},
		{name: "403", err: &api.APIError{StatusCode: http.StatusForbidden, Message: "nope"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isNotFound(tt.err); got != tt.want {
				t.Fatalf("isNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAPIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "generic", err: errors.New("plain"), want: "plain"},
		{
			name: "api with message",
			err:  &api.APIError{StatusCode: http.StatusForbidden, Message: "Partner feature required"},
			want: "AhaSend API error (HTTP 403): Partner feature required",
		},
		{
			name: "api empty message",
			err:  &api.APIError{StatusCode: http.StatusTooManyRequests, Message: ""},
			want: "AhaSend API error (HTTP 429): Too Many Requests",
		},
		{
			name: "api empty message unknown status",
			err:  &api.APIError{StatusCode: 599, Message: ""},
			want: "AhaSend API error (HTTP 599): Unknown HTTP status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatAPIError(tt.err)
			if got != tt.want {
				t.Fatalf("formatAPIError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAhaSendClientInvalidAccount(t *testing.T) {
	t.Parallel()
	_, err := newAhaSendClient("key", "not-uuid", "https://api.ahasend.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigureEndpoint(t *testing.T) {
	t.Parallel()
	client := api.NewAPIClient(api.WithAPIKey("test-key"))
	if err := configureEndpoint(client, "not a url"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if err := configureEndpoint(client, "https://api.example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cfg := client.GetConfig()
	if cfg.Host != "api.example.com" || cfg.Scheme != "https" {
		t.Fatalf("unexpected config host=%q scheme=%q", cfg.Host, cfg.Scheme)
	}
}

func TestBoolPtr(t *testing.T) {
	t.Parallel()
	p := boolPtr(true)
	if p == nil || !*p {
		t.Fatal("boolPtr(true) failed")
	}
	p = boolPtr(false)
	if p == nil || *p {
		t.Fatal("boolPtr(false) failed")
	}
}
