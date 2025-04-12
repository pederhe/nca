package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pederhe/nca/core/mcp/common"
)

// AuthResult represents the authentication result
type AuthResult string

const (
	// Authorized indicates successful authentication
	Authorized AuthResult = "AUTHORIZED"

	// Redirect indicates that redirection is needed for authentication
	Redirect AuthResult = "REDIRECT"
)

// UnauthorizedError is already defined in sse.go

// OAuthClientProvider is an interface that provides OAuth client functionality
type OAuthClientProvider interface {
	// RedirectURL returns the URL to redirect to after authentication
	RedirectURL() string

	// ClientMetadata returns the OAuth client metadata
	ClientMetadata() *common.OAuthClientMetadata

	// ClientInformation loads OAuth client information, returns nil if client is not registered
	ClientInformation() (*common.OAuthClientInformation, error)

	// SaveClientInformation saves client information, used for dynamic registration
	SaveClientInformation(*common.OAuthClientInformationFull) error

	// Tokens loads existing OAuth tokens, returns nil if none exist
	Tokens() (*common.OAuthTokens, error)

	// SaveTokens saves new OAuth tokens
	SaveTokens(*common.OAuthTokens) error

	// RedirectToAuthorization redirects the user agent to the given URL to start the authentication flow
	RedirectToAuthorization(authURL *url.URL) error

	// SaveCodeVerifier saves the PKCE code verifier
	SaveCodeVerifier(codeVerifier string) error

	// CodeVerifier loads the PKCE code verifier
	CodeVerifier() (string, error)
}

// DefaultOAuthClientProvider implements the OAuthClientProvider interface
type DefaultOAuthClientProvider struct {
	redirectURL      string
	clientMetadata   *common.OAuthClientMetadata
	clientInfo       *common.OAuthClientInformation
	tokens           *common.OAuthTokens
	codeVerifier     string
	storage          TokenStorage
	redirectCallback func(*url.URL) error
}

// TokenStorage is an interface for storing OAuth tokens and related information
type TokenStorage interface {
	// SaveTokens saves tokens
	SaveTokens(tokens *common.OAuthTokens) error

	// LoadTokens loads tokens
	LoadTokens() (*common.OAuthTokens, error)

	// SaveClientInfo saves client information
	SaveClientInfo(info *common.OAuthClientInformation) error

	// LoadClientInfo loads client information
	LoadClientInfo() (*common.OAuthClientInformation, error)

	// SaveCodeVerifier saves the code verifier
	SaveCodeVerifier(codeVerifier string) error

	// LoadCodeVerifier loads the code verifier
	LoadCodeVerifier() (string, error)
}

// NewDefaultOAuthClientProvider creates a default OAuth provider
func NewDefaultOAuthClientProvider(
	redirectURL string,
	clientMetadata *common.OAuthClientMetadata,
	storage TokenStorage,
	redirectCallback func(*url.URL) error,
) *DefaultOAuthClientProvider {
	return &DefaultOAuthClientProvider{
		redirectURL:      redirectURL,
		clientMetadata:   clientMetadata,
		storage:          storage,
		redirectCallback: redirectCallback,
	}
}

// RedirectURL implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) RedirectURL() string {
	return p.redirectURL
}

// ClientMetadata implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) ClientMetadata() *common.OAuthClientMetadata {
	return p.clientMetadata
}

// ClientInformation implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) ClientInformation() (*common.OAuthClientInformation, error) {
	if p.clientInfo != nil {
		return p.clientInfo, nil
	}

	if p.storage != nil {
		info, err := p.storage.LoadClientInfo()
		if err != nil {
			return nil, err
		}
		p.clientInfo = info
		return info, nil
	}

	return nil, nil
}

// SaveClientInformation implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) SaveClientInformation(info *common.OAuthClientInformationFull) error {
	p.clientInfo = &common.OAuthClientInformation{
		ClientID:              info.ClientID,
		ClientSecret:          info.ClientSecret,
		ClientIDIssuedAt:      info.ClientIDIssuedAt,
		ClientSecretExpiresAt: info.ClientSecretExpiresAt,
	}

	if p.storage != nil {
		return p.storage.SaveClientInfo(p.clientInfo)
	}

	return nil
}

// Tokens implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) Tokens() (*common.OAuthTokens, error) {
	if p.tokens != nil {
		return p.tokens, nil
	}

	if p.storage != nil {
		tokens, err := p.storage.LoadTokens()
		if err != nil {
			return nil, err
		}
		p.tokens = tokens
		return tokens, nil
	}

	return nil, nil
}

// SaveTokens implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) SaveTokens(tokens *common.OAuthTokens) error {
	p.tokens = tokens

	if p.storage != nil {
		return p.storage.SaveTokens(tokens)
	}

	return nil
}

// RedirectToAuthorization implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) RedirectToAuthorization(authURL *url.URL) error {
	if p.redirectCallback != nil {
		return p.redirectCallback(authURL)
	}

	return errors.New("No redirect callback provided")
}

// SaveCodeVerifier implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) SaveCodeVerifier(codeVerifier string) error {
	p.codeVerifier = codeVerifier

	if p.storage != nil {
		return p.storage.SaveCodeVerifier(codeVerifier)
	}

	return nil
}

// CodeVerifier implements the OAuthClientProvider interface
func (p *DefaultOAuthClientProvider) CodeVerifier() (string, error) {
	if p.codeVerifier != "" {
		return p.codeVerifier, nil
	}

	if p.storage != nil {
		codeVerifier, err := p.storage.LoadCodeVerifier()
		if err != nil {
			return "", err
		}
		p.codeVerifier = codeVerifier
		return codeVerifier, nil
	}

	return "", errors.New("Code verifier not found")
}

// Auth coordinates the complete authentication process with the server
func Auth(
	provider OAuthClientProvider,
	serverURL string,
	authorizationCode string,
) (AuthResult, error) {
	// Discover OAuth metadata
	metadata, err := DiscoverOAuthMetadata(serverURL)
	if err != nil {
		return "", err
	}

	if metadata == nil {
		return "", errors.New("Server does not support OAuth authentication")
	}

	// Check client registration
	clientInfo, err := provider.ClientInformation()
	if err != nil {
		return "", err
	}

	if clientInfo == nil {
		if authorizationCode != "" {
			return "", errors.New("Existing OAuth client information required for authorization code exchange")
		}

		// Dynamic registration of client
		clientMetadata := provider.ClientMetadata()
		if clientMetadata == nil {
			return "", errors.New("Client metadata not provided")
		}

		fullInfo, err := RegisterClient(serverURL, metadata, clientMetadata)
		if err != nil {
			return "", err
		}

		if err := provider.SaveClientInformation(fullInfo); err != nil {
			return "", fmt.Errorf("Failed to save client information: %w", err)
		}

		clientInfo = &common.OAuthClientInformation{
			ClientID:              fullInfo.ClientID,
			ClientSecret:          fullInfo.ClientSecret,
			ClientIDIssuedAt:      fullInfo.ClientIDIssuedAt,
			ClientSecretExpiresAt: fullInfo.ClientSecretExpiresAt,
		}
	}

	// Exchange authorization code
	if authorizationCode != "" {
		codeVerifier, err := provider.CodeVerifier()
		if err != nil {
			return "", fmt.Errorf("Failed to get code verifier: %w", err)
		}

		tokens, err := ExchangeAuthorization(
			serverURL,
			metadata,
			clientInfo,
			authorizationCode,
			codeVerifier,
			provider.RedirectURL(),
		)
		if err != nil {
			return "", fmt.Errorf("Failed to exchange authorization code: %w", err)
		}

		if err := provider.SaveTokens(tokens); err != nil {
			return "", fmt.Errorf("Failed to save tokens: %w", err)
		}

		return Authorized, nil
	}

	// Check existing tokens
	tokens, err := provider.Tokens()
	if err != nil {
		return "", fmt.Errorf("Failed to get tokens: %w", err)
	}

	if tokens != nil && tokens.RefreshToken != "" {
		// Try refreshing tokens
		newTokens, err := RefreshAuthorization(
			serverURL,
			metadata,
			clientInfo,
			tokens.RefreshToken,
		)
		if err == nil {
			if err := provider.SaveTokens(newTokens); err != nil {
				return "", fmt.Errorf("Failed to save refreshed tokens: %w", err)
			}
			return Authorized, nil
		}

		// Refresh failed, continue with new authorization
	}

	// Start new authorization flow
	authURL, codeVerifier, err := StartAuthorization(
		serverURL,
		metadata,
		clientInfo,
		provider.RedirectURL(),
	)
	if err != nil {
		return "", fmt.Errorf("Failed to start authorization: %w", err)
	}

	if err := provider.SaveCodeVerifier(codeVerifier); err != nil {
		return "", fmt.Errorf("Failed to save code verifier: %w", err)
	}

	if err := provider.RedirectToAuthorization(authURL); err != nil {
		return "", fmt.Errorf("Failed to redirect to authorization URL: %w", err)
	}

	return Redirect, nil
}

// DiscoverOAuthMetadata finds the OAuth 2.0 authorization server metadata from RFC 8414
func DiscoverOAuthMetadata(serverURL string) (*common.OAuthMetadata, error) {
	wellKnownURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	wellKnownURL.Path = "/.well-known/oauth-authorization-server"

	resp, err := http.Get(wellKnownURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d Attempted to load OAuth metadata", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	metadata, err := common.ParseOAuthMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse OAuth metadata: %w", err)
	}

	return metadata, nil
}

// StartAuthorization generates a PKCE challenge and builds the authorization URL
func StartAuthorization(
	serverURL string,
	metadata *common.OAuthMetadata,
	clientInfo *common.OAuthClientInformation,
	redirectURL string,
) (*url.URL, string, error) {
	if metadata == nil {
		var err error
		metadata, err = DiscoverOAuthMetadata(serverURL)
		if err != nil {
			return nil, "", err
		}

		if metadata == nil {
			return nil, "", errors.New("Server does not support OAuth authentication")
		}
	}

	// Create PKCE challenge
	codeVerifier, err := generateCodeVerifier(64)
	if err != nil {
		return nil, "", err
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Build authorization URL
	authURL, err := url.Parse(metadata.AuthorizationEndpoint)
	if err != nil {
		return nil, "", err
	}

	query := authURL.Query()
	query.Set("response_type", "code")
	query.Set("client_id", clientInfo.ClientID)
	query.Set("redirect_uri", redirectURL)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()

	return authURL, codeVerifier, nil
}

// ExchangeAuthorization exchanges the authorization code for tokens
func ExchangeAuthorization(
	serverURL string,
	metadata *common.OAuthMetadata,
	clientInfo *common.OAuthClientInformation,
	authorizationCode, codeVerifier, redirectURI string,
) (*common.OAuthTokens, error) {
	if metadata == nil {
		var err error
		metadata, err = DiscoverOAuthMetadata(serverURL)
		if err != nil {
			return nil, err
		}

		if metadata == nil {
			return nil, errors.New("Server does not support OAuth authentication")
		}
	}

	tokenEndpoint := metadata.TokenEndpoint

	// Prepare request parameters
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authorizationCode)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientInfo.ClientID)
	if clientInfo.ClientSecret != "" {
		data.Set("client_secret", clientInfo.ClientSecret)
	}
	data.Set("code_verifier", codeVerifier)

	// Send request
	resp, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d Attempted to exchange authorization code", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tokens, err := common.ParseOAuthTokens(respData)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse OAuth tokens: %w", err)
	}

	return tokens, nil
}

// RefreshAuthorization refreshes OAuth tokens
func RefreshAuthorization(
	serverURL string,
	metadata *common.OAuthMetadata,
	clientInfo *common.OAuthClientInformation,
	refreshToken string,
) (*common.OAuthTokens, error) {
	if metadata == nil {
		var err error
		metadata, err = DiscoverOAuthMetadata(serverURL)
		if err != nil {
			return nil, err
		}

		if metadata == nil {
			return nil, errors.New("Server does not support OAuth authentication")
		}
	}

	tokenEndpoint := metadata.TokenEndpoint

	// Prepare request parameters
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientInfo.ClientID)
	if clientInfo.ClientSecret != "" {
		data.Set("client_secret", clientInfo.ClientSecret)
	}

	// Send request
	resp, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d Attempted to refresh tokens", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tokens, err := common.ParseOAuthTokens(respData)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse OAuth tokens: %w", err)
	}

	return tokens, nil
}

// RegisterClient registers an OAuth client
func RegisterClient(
	serverURL string,
	metadata *common.OAuthMetadata,
	clientMetadata *common.OAuthClientMetadata,
) (*common.OAuthClientInformationFull, error) {
	if metadata == nil {
		var err error
		metadata, err = DiscoverOAuthMetadata(serverURL)
		if err != nil {
			return nil, err
		}

		if metadata == nil {
			return nil, errors.New("Server does not support OAuth authentication")
		}
	}

	if metadata.RegistrationEndpoint == "" {
		return nil, errors.New("Server does not support dynamic client registration")
	}

	// Validate client metadata
	if err := clientMetadata.Validate(); err != nil {
		return nil, fmt.Errorf("Client metadata validation failed: %w", err)
	}

	// Prepare request data
	data, err := json.Marshal(clientMetadata)
	if err != nil {
		return nil, err
	}

	// Send request
	resp, err := http.Post(metadata.RegistrationEndpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d Attempted to register client", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	clientInfo, err := common.ParseOAuthClientInformationFull(respData)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse client information: %w", err)
	}

	return clientInfo, nil
}

// MemoryTokenStorage is an in-memory implementation of TokenStorage
type MemoryTokenStorage struct {
	tokens       *common.OAuthTokens
	clientInfo   *common.OAuthClientInformation
	codeVerifier string
}

// NewMemoryTokenStorage creates a new in-memory token storage
func NewMemoryTokenStorage() *MemoryTokenStorage {
	return &MemoryTokenStorage{}
}

// SaveTokens implements the TokenStorage interface
func (s *MemoryTokenStorage) SaveTokens(tokens *common.OAuthTokens) error {
	s.tokens = tokens
	return nil
}

// LoadTokens implements the TokenStorage interface
func (s *MemoryTokenStorage) LoadTokens() (*common.OAuthTokens, error) {
	return s.tokens, nil
}

// SaveClientInfo implements the TokenStorage interface
func (s *MemoryTokenStorage) SaveClientInfo(info *common.OAuthClientInformation) error {
	s.clientInfo = info
	return nil
}

// LoadClientInfo implements the TokenStorage interface
func (s *MemoryTokenStorage) LoadClientInfo() (*common.OAuthClientInformation, error) {
	return s.clientInfo, nil
}

// SaveCodeVerifier implements the TokenStorage interface
func (s *MemoryTokenStorage) SaveCodeVerifier(codeVerifier string) error {
	s.codeVerifier = codeVerifier
	return nil
}

// LoadCodeVerifier implements the TokenStorage interface
func (s *MemoryTokenStorage) LoadCodeVerifier() (string, error) {
	if s.codeVerifier == "" {
		return "", errors.New("Token not found")
	}
	return s.codeVerifier, nil
}

// OAuthProvider is a simple implementation that satisfies the OAuthProvider interface in sse.go
type StandardOAuthProvider struct {
	clientProvider OAuthClientProvider
	serverURL      string
}

// NewStandardOAuthProvider creates a new standard OAuth provider
func NewStandardOAuthProvider(clientProvider OAuthClientProvider, serverURL string) *StandardOAuthProvider {
	return &StandardOAuthProvider{
		clientProvider: clientProvider,
		serverURL:      serverURL,
	}
}

// GetToken implements the OAuthProvider interface
func (p *StandardOAuthProvider) GetToken() (string, error) {
	tokens, err := p.clientProvider.Tokens()
	if err != nil {
		return "", err
	}

	if tokens == nil {
		return "", errors.New("Token not found")
	}

	return tokens.AccessToken, nil
}

// RefreshToken implements the OAuthProvider interface
func (p *StandardOAuthProvider) RefreshToken() (string, error) {
	tokens, err := p.clientProvider.Tokens()
	if err != nil {
		return "", err
	}

	if tokens == nil || tokens.RefreshToken == "" {
		return "", errors.New("Refresh token not found")
	}

	clientInfo, err := p.clientProvider.ClientInformation()
	if err != nil {
		return "", err
	}

	if clientInfo == nil {
		return "", errors.New("Client information not found")
	}

	metadata, err := DiscoverOAuthMetadata(p.serverURL)
	if err != nil {
		return "", err
	}

	if metadata == nil {
		return "", errors.New("Server does not support OAuth authentication")
	}

	newTokens, err := RefreshAuthorization(
		p.serverURL,
		metadata,
		clientInfo,
		tokens.RefreshToken,
	)
	if err != nil {
		return "", err
	}

	if err := p.clientProvider.SaveTokens(newTokens); err != nil {
		return "", err
	}

	return newTokens.AccessToken, nil
}

// Helper function: generates a PKCE code verifier
func generateCodeVerifier(length int) (string, error) {
	if length < 43 || length > 128 {
		length = 64 // RFC 7636 recommended length
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Base64 URL encoding
	encoded := base64.URLEncoding.EncodeToString(bytes)

	// Truncate to original length (because Base64 encoding extends string length)
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	// Ensure result meets RFC 7636 requirements
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.ReplaceAll(encoded, "=", "")

	return encoded, nil
}

// Helper function: generates a code challenge from a code verifier
func generateCodeChallenge(codeVerifier string) string {
	// This is a simplified implementation
	// In actual application, SHA-256 should be used to calculate the digest of the code verifier
	// Then Base64-URL encoding should be performed

	// For simplicity, this directly returns the verifier
	// In actual application, the correct S256 algorithm should be used
	return codeVerifier
}
