package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// OAuthMetadata represents RFC 8414 OAuth 2.0 Authorization Server Metadata
type OAuthMetadata struct {
	Issuer                                             string   `json:"issuer"`
	AuthorizationEndpoint                              string   `json:"authorization_endpoint"`
	TokenEndpoint                                      string   `json:"token_endpoint"`
	RegistrationEndpoint                               string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                                    []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported                             []string `json:"response_types_supported"`
	ResponseModesSupported                             []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported                                []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported                  []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	TokenEndpointAuthSigningAlgValuesSupported         []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	ServiceDocumentation                               string   `json:"service_documentation,omitempty"`
	RevocationEndpoint                                 string   `json:"revocation_endpoint,omitempty"`
	RevocationEndpointAuthMethodsSupported             []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	RevocationEndpointAuthSigningAlgValuesSupported    []string `json:"revocation_endpoint_auth_signing_alg_values_supported,omitempty"`
	IntrospectionEndpoint                              string   `json:"introspection_endpoint,omitempty"`
	IntrospectionEndpointAuthMethodsSupported          []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	IntrospectionEndpointAuthSigningAlgValuesSupported []string `json:"introspection_endpoint_auth_signing_alg_values_supported,omitempty"`
	CodeChallengeMethodsSupported                      []string `json:"code_challenge_methods_supported,omitempty"`
}

// OAuthTokens represents OAuth 2.1 token response
type OAuthTokens struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// OAuthErrorResponse represents OAuth 2.1 error response
type OAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// OAuthClientMetadata represents RFC 7591 OAuth 2.0 Dynamic Client Registration metadata
type OAuthClientMetadata struct {
	RedirectURIs            []string        `json:"redirect_uris"`
	TokenEndpointAuthMethod string          `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string        `json:"grant_types,omitempty"`
	ResponseTypes           []string        `json:"response_types,omitempty"`
	ClientName              string          `json:"client_name,omitempty"`
	ClientURI               string          `json:"client_uri,omitempty"`
	LogoURI                 string          `json:"logo_uri,omitempty"`
	Scope                   string          `json:"scope,omitempty"`
	Contacts                []string        `json:"contacts,omitempty"`
	TOSURI                  string          `json:"tos_uri,omitempty"`
	PolicyURI               string          `json:"policy_uri,omitempty"`
	JWKSURI                 string          `json:"jwks_uri,omitempty"`
	JWKS                    json.RawMessage `json:"jwks,omitempty"`
	SoftwareID              string          `json:"software_id,omitempty"`
	SoftwareVersion         string          `json:"software_version,omitempty"`
}

// Validate validates the client metadata
func (cm *OAuthClientMetadata) Validate() error {
	if len(cm.RedirectURIs) == 0 {
		return errors.New("redirect_uris is required")
	}

	for _, uri := range cm.RedirectURIs {
		if _, err := url.Parse(uri); err != nil {
			return fmt.Errorf("invalid redirect_uri: %s", uri)
		}
	}

	return nil
}

// OAuthClientInformation represents RFC 7591 OAuth 2.0 Dynamic Client Registration client information
type OAuthClientInformation struct {
	ClientID              string `json:"client_id"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientIDIssuedAt      int64  `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt int64  `json:"client_secret_expires_at,omitempty"`
}

// IsClientSecretExpired checks if the client secret has expired
func (ci *OAuthClientInformation) IsClientSecretExpired() bool {
	if ci.ClientSecretExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > ci.ClientSecretExpiresAt
}

// OAuthClientInformationFull represents RFC 7591 OAuth 2.0 Dynamic Client Registration full response
type OAuthClientInformationFull struct {
	OAuthClientMetadata
	OAuthClientInformation
}

// OAuthClientRegistrationError represents RFC 7591 OAuth 2.0 Dynamic Client Registration error response
type OAuthClientRegistrationError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// OAuthTokenRevocationRequest represents RFC 7009 OAuth 2.0 Token Revocation request
type OAuthTokenRevocationRequest struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint,omitempty"`
}

// ParseOAuthMetadata parses OAuth metadata from JSON
func ParseOAuthMetadata(data []byte) (*OAuthMetadata, error) {
	var metadata OAuthMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	// Validate required fields
	if metadata.Issuer == "" {
		return nil, errors.New("issuer is required")
	}
	if metadata.AuthorizationEndpoint == "" {
		return nil, errors.New("authorization_endpoint is required")
	}
	if metadata.TokenEndpoint == "" {
		return nil, errors.New("token_endpoint is required")
	}
	if len(metadata.ResponseTypesSupported) == 0 {
		return nil, errors.New("response_types_supported is required")
	}

	return &metadata, nil
}

// ParseOAuthTokens parses OAuth tokens from JSON
func ParseOAuthTokens(data []byte) (*OAuthTokens, error) {
	var tokens OAuthTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, err
	}

	// Validate required fields
	if tokens.AccessToken == "" {
		return nil, errors.New("access_token is required")
	}
	if tokens.TokenType == "" {
		return nil, errors.New("token_type is required")
	}

	return &tokens, nil
}

// ParseOAuthErrorResponse parses OAuth error response from JSON
func ParseOAuthErrorResponse(data []byte) (*OAuthErrorResponse, error) {
	var errorResponse OAuthErrorResponse
	if err := json.Unmarshal(data, &errorResponse); err != nil {
		return nil, err
	}

	// Validate required fields
	if errorResponse.Error == "" {
		return nil, errors.New("error is required")
	}

	return &errorResponse, nil
}

// ParseOAuthClientMetadata parses OAuth client metadata from JSON
func ParseOAuthClientMetadata(data []byte) (*OAuthClientMetadata, error) {
	var metadata OAuthClientMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	if err := metadata.Validate(); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// ParseOAuthClientInformation parses OAuth client information from JSON
func ParseOAuthClientInformation(data []byte) (*OAuthClientInformation, error) {
	var info OAuthClientInformation
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	// Validate required fields
	if info.ClientID == "" {
		return nil, errors.New("client_id is required")
	}

	return &info, nil
}

// ParseOAuthClientInformationFull parses full OAuth client information from JSON
func ParseOAuthClientInformationFull(data []byte) (*OAuthClientInformationFull, error) {
	var info OAuthClientInformationFull
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	// Validate required fields
	if info.ClientID == "" {
		return nil, errors.New("client_id is required")
	}

	if err := info.OAuthClientMetadata.Validate(); err != nil {
		return nil, err
	}

	return &info, nil
}

// ParseOAuthClientRegistrationError parses OAuth client registration error from JSON
func ParseOAuthClientRegistrationError(data []byte) (*OAuthClientRegistrationError, error) {
	var errorResponse OAuthClientRegistrationError
	if err := json.Unmarshal(data, &errorResponse); err != nil {
		return nil, err
	}

	// Validate required fields
	if errorResponse.Error == "" {
		return nil, errors.New("error is required")
	}

	return &errorResponse, nil
}

// ParseOAuthTokenRevocationRequest parses OAuth token revocation request from JSON
func ParseOAuthTokenRevocationRequest(data []byte) (*OAuthTokenRevocationRequest, error) {
	var request OAuthTokenRevocationRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}

	// Validate required fields
	if request.Token == "" {
		return nil, errors.New("token is required")
	}

	return &request, nil
}
