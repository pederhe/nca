package common

import (
	"testing"
	"time"
)

func TestOAuthClientMetadata_Validate(t *testing.T) {
	// Test valid metadata
	validMetadata := &OAuthClientMetadata{
		RedirectURIs: []string{"https://example.com/callback"},
		ClientName:   "Test Client",
	}
	if err := validMetadata.Validate(); err != nil {
		t.Errorf("Expected valid metadata to pass validation, got error: %v", err)
	}

	// Test missing redirect_uris
	invalidMetadata := &OAuthClientMetadata{
		ClientName: "Test Client",
	}
	if err := invalidMetadata.Validate(); err == nil {
		t.Error("Expected error for metadata without redirect_uris, got nil")
	}

	// Test invalid redirect URI - using an explicitly invalid URL, such as one without a protocol
	invalidURIMetadata := &OAuthClientMetadata{
		RedirectURIs: []string{"://invalid-url"},
		ClientName:   "Test Client",
	}
	if err := invalidURIMetadata.Validate(); err == nil {
		t.Error("Expected error for metadata with invalid redirect URI, got nil")
	}
}

func TestOAuthClientInformation_IsClientSecretExpired(t *testing.T) {
	// Test non-expiring client secret
	nonExpiringClient := &OAuthClientInformation{
		ClientID:              "client_id",
		ClientSecret:          "client_secret",
		ClientSecretExpiresAt: 0, // 0 means never expires
	}
	if nonExpiringClient.IsClientSecretExpired() {
		t.Error("Expected non-expiring client secret to not be expired")
	}

	// Test expired client secret
	pastTime := time.Now().Add(-1 * time.Hour).Unix()
	expiredClient := &OAuthClientInformation{
		ClientID:              "client_id",
		ClientSecret:          "client_secret",
		ClientSecretExpiresAt: pastTime,
	}
	if !expiredClient.IsClientSecretExpired() {
		t.Error("Expected expired client secret to be expired")
	}

	// Test non-expired client secret
	futureTime := time.Now().Add(1 * time.Hour).Unix()
	validClient := &OAuthClientInformation{
		ClientID:              "client_id",
		ClientSecret:          "client_secret",
		ClientSecretExpiresAt: futureTime,
	}
	if validClient.IsClientSecretExpired() {
		t.Error("Expected non-expired client secret to not be expired")
	}
}

func TestParseOAuthMetadata(t *testing.T) {
	// Test valid metadata JSON
	validJSON := []byte(`{
		"issuer": "https://auth.example.com",
		"authorization_endpoint": "https://auth.example.com/authorize",
		"token_endpoint": "https://auth.example.com/token",
		"response_types_supported": ["code", "token"]
	}`)
	metadata, err := ParseOAuthMetadata(validJSON)
	if err != nil {
		t.Errorf("Expected valid JSON to parse without error, got: %v", err)
	}
	if metadata.Issuer != "https://auth.example.com" {
		t.Errorf("Expected issuer to be 'https://auth.example.com', got '%s'", metadata.Issuer)
	}

	// Test missing required fields
	missingIssuerJSON := []byte(`{
		"authorization_endpoint": "https://auth.example.com/authorize",
		"token_endpoint": "https://auth.example.com/token",
		"response_types_supported": ["code", "token"]
	}`)
	_, err = ParseOAuthMetadata(missingIssuerJSON)
	if err == nil {
		t.Error("Expected error for metadata missing issuer, got nil")
	}

	// Test invalid JSON
	invalidJSON := []byte(`{not-valid-json}`)
	_, err = ParseOAuthMetadata(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseOAuthTokens(t *testing.T) {
	// Test valid tokens JSON
	validJSON := []byte(`{
		"access_token": "access_token_value",
		"token_type": "bearer",
		"expires_in": 3600,
		"refresh_token": "refresh_token_value"
	}`)
	tokens, err := ParseOAuthTokens(validJSON)
	if err != nil {
		t.Errorf("Expected valid JSON to parse without error, got: %v", err)
	}
	if tokens.AccessToken != "access_token_value" {
		t.Errorf("Expected access_token to be 'access_token_value', got '%s'", tokens.AccessToken)
	}

	// Test missing required fields
	missingTokenTypeJSON := []byte(`{
		"access_token": "access_token_value"
	}`)
	_, err = ParseOAuthTokens(missingTokenTypeJSON)
	if err == nil {
		t.Error("Expected error for tokens missing token_type, got nil")
	}

	// Test invalid JSON
	invalidJSON := []byte(`{not-valid-json}`)
	_, err = ParseOAuthTokens(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseOAuthErrorResponse(t *testing.T) {
	// Test valid error response JSON
	validJSON := []byte(`{
		"error": "invalid_request",
		"error_description": "The request is missing a required parameter"
	}`)
	errorResponse, err := ParseOAuthErrorResponse(validJSON)
	if err != nil {
		t.Errorf("Expected valid JSON to parse without error, got: %v", err)
	}
	if errorResponse.Error != "invalid_request" {
		t.Errorf("Expected error to be 'invalid_request', got '%s'", errorResponse.Error)
	}

	// Test missing required fields
	missingErrorJSON := []byte(`{
		"error_description": "The request is missing a required parameter"
	}`)
	_, err = ParseOAuthErrorResponse(missingErrorJSON)
	if err == nil {
		t.Error("Expected error for response missing error field, got nil")
	}

	// Test invalid JSON
	invalidJSON := []byte(`{not-valid-json}`)
	_, err = ParseOAuthErrorResponse(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestParseOAuthClientMetadata(t *testing.T) {
	// Test valid client metadata JSON
	validJSON := []byte(`{
		"redirect_uris": ["https://client.example.com/callback"],
		"client_name": "Example Client",
		"scope": "read write"
	}`)
	metadata, err := ParseOAuthClientMetadata(validJSON)
	if err != nil {
		t.Errorf("Expected valid JSON to parse without error, got: %v", err)
	}
	if metadata.ClientName != "Example Client" {
		t.Errorf("Expected client_name to be 'Example Client', got '%s'", metadata.ClientName)
	}

	// Test invalid JSON
	invalidJSON := []byte(`{not-valid-json}`)
	_, err = ParseOAuthClientMetadata(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
