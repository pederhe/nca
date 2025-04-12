package client

import (
	"errors"
	"net/url"
	"testing"

	"github.com/pederhe/nca/core/mcp/common"
	"github.com/stretchr/testify/assert"
)

func TestDefaultOAuthClientProvider(t *testing.T) {
	// Create memory storage
	storage := NewMemoryTokenStorage()

	// Create test metadata
	metadata := &common.OAuthClientMetadata{
		ClientName:              "TestClient",
		RedirectURIs:            []string{"http://localhost:8000/callback"},
		TokenEndpointAuthMethod: "none",
	}

	// Create redirect callback
	var redirectURL *url.URL
	redirectCallback := func(authURL *url.URL) error {
		redirectURL = authURL
		return nil
	}

	// Create OAuth provider
	provider := NewDefaultOAuthClientProvider(
		"http://localhost:8000/callback",
		metadata,
		storage,
		redirectCallback,
	)

	// Test RedirectURL
	assert.Equal(t, "http://localhost:8000/callback", provider.RedirectURL(), "RedirectURL should match")

	// Test ClientMetadata
	assert.Equal(t, metadata, provider.ClientMetadata(), "ClientMetadata should match")

	// Test that Tokens and ClientInformation are nil initially
	tokens, err := provider.Tokens()
	assert.NoError(t, err, "Getting tokens should not error")
	assert.Nil(t, tokens, "Tokens should be nil initially")

	clientInfo, err := provider.ClientInformation()
	assert.NoError(t, err, "Getting client info should not error")
	assert.Nil(t, clientInfo, "Client info should be nil initially")

	// Test SaveTokens
	testTokens := &common.OAuthTokens{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "bearer",
		ExpiresIn:    3600,
	}

	err = provider.SaveTokens(testTokens)
	assert.NoError(t, err, "SaveTokens should not error")

	// Verify Tokens are saved
	savedTokens, err := provider.Tokens()
	assert.NoError(t, err, "Getting tokens should not error")
	assert.Equal(t, testTokens, savedTokens, "Saved tokens should match")

	// Test SaveClientInformation
	testClientInfo := &common.OAuthClientInformationFull{
		OAuthClientInformation: common.OAuthClientInformation{
			ClientID:              "test-client-id",
			ClientSecret:          "test-client-secret",
			ClientIDIssuedAt:      1234567890,
			ClientSecretExpiresAt: 1234567890 + 3600,
		},
	}

	err = provider.SaveClientInformation(testClientInfo)
	assert.NoError(t, err, "SaveClientInformation should not error")

	// Verify ClientInformation is saved
	savedClientInfo, err := provider.ClientInformation()
	assert.NoError(t, err, "Getting client info should not error")
	assert.Equal(t, "test-client-id", savedClientInfo.ClientID, "Client ID should match")
	assert.Equal(t, "test-client-secret", savedClientInfo.ClientSecret, "Client secret should match")

	// Test SaveCodeVerifier and CodeVerifier
	err = provider.SaveCodeVerifier("test-code-verifier")
	assert.NoError(t, err, "SaveCodeVerifier should not error")

	codeVerifier, err := provider.CodeVerifier()
	assert.NoError(t, err, "CodeVerifier should not error")
	assert.Equal(t, "test-code-verifier", codeVerifier, "Code verifier should match")

	// Test RedirectToAuthorization
	authURL, err := url.Parse("https://example.com/auth")
	assert.NoError(t, err, "Parsing URL should not error")

	err = provider.RedirectToAuthorization(authURL)
	assert.NoError(t, err, "RedirectToAuthorization should not error")
	assert.Equal(t, authURL, redirectURL, "Redirect URL should match")
}

func TestMemoryTokenStorage(t *testing.T) {
	// Create memory storage
	storage := NewMemoryTokenStorage()

	// Test that all get operations return nil initially
	tokens, err := storage.LoadTokens()
	assert.NoError(t, err, "LoadTokens should not error")
	assert.Nil(t, tokens, "Tokens should be nil initially")

	clientInfo, err := storage.LoadClientInfo()
	assert.NoError(t, err, "LoadClientInfo should not error")
	assert.Nil(t, clientInfo, "Client info should be nil initially")

	codeVerifier, err := storage.LoadCodeVerifier()
	assert.Equal(t, "", codeVerifier, "Code verifier should be empty initially")

	// Test saving and loading tokens
	testTokens := &common.OAuthTokens{
		AccessToken:  "memory-access-token",
		RefreshToken: "memory-refresh-token",
		TokenType:    "bearer",
		ExpiresIn:    3600,
	}

	err = storage.SaveTokens(testTokens)
	assert.NoError(t, err, "SaveTokens should not error")

	loadedTokens, err := storage.LoadTokens()
	assert.NoError(t, err, "LoadTokens should not error")
	assert.Equal(t, testTokens, loadedTokens, "Loaded tokens should match saved tokens")

	// Test saving and loading client information
	testClientInfo := &common.OAuthClientInformation{
		ClientID:              "memory-client-id",
		ClientSecret:          "memory-client-secret",
		ClientIDIssuedAt:      1234567890,
		ClientSecretExpiresAt: 1234567890 + 3600,
	}

	err = storage.SaveClientInfo(testClientInfo)
	assert.NoError(t, err, "SaveClientInfo should not error")

	loadedClientInfo, err := storage.LoadClientInfo()
	assert.NoError(t, err, "LoadClientInfo should not error")
	assert.Equal(t, testClientInfo, loadedClientInfo, "Loaded client info should match saved client info")

	// Test saving and loading code verifier
	err = storage.SaveCodeVerifier("memory-code-verifier")
	assert.NoError(t, err, "SaveCodeVerifier should not error")

	loadedCodeVerifier, err := storage.LoadCodeVerifier()
	assert.NoError(t, err, "LoadCodeVerifier should not error")
	assert.Equal(t, "memory-code-verifier", loadedCodeVerifier, "Loaded code verifier should match saved code verifier")
}

func TestStandardOAuthProvider(t *testing.T) {
	// Create mock OAuthClientProvider
	mockProvider := &MockOAuthClientProvider{
		redirectURL: "http://example.com/callback",
		tokens: &common.OAuthTokens{
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-refresh-token",
			TokenType:    "bearer",
			ExpiresIn:    3600,
		},
		clientInfo: &common.OAuthClientInformation{
			ClientID:     "mock-client-id",
			ClientSecret: "mock-client-secret",
		},
		codeVerifier: "mock-code-verifier",
	}

	// Create StandardOAuthProvider
	provider := &StandardOAuthProviderMock{
		clientProvider: mockProvider,
		serverURL:      "https://example.com",
	}

	// Test GetToken
	token, err := provider.GetToken()
	assert.NoError(t, err, "GetToken should not error")
	assert.Equal(t, "mock-access-token", token, "Access token should match")

	// Simulate token expiration scenario
	mockProvider.shouldRefresh = true
	mockProvider.tokens.AccessToken = "expired-token"

	// Test GetToken automatically refreshes expired token
	token, err = provider.GetToken()
	assert.NoError(t, err, "GetToken should not error even when refreshing")
	assert.Equal(t, "refreshed-access-token", token, "Should get refreshed access token")

	// Test RefreshToken
	token, err = provider.RefreshToken()
	assert.NoError(t, err, "RefreshToken should not error")
	assert.Equal(t, "refreshed-access-token", token, "Should get refreshed access token")

	// Test refresh failure scenario
	mockProvider.shouldFailRefresh = true

	_, err = provider.RefreshToken()
	assert.Error(t, err, "RefreshToken should error when refresh fails")
	assert.Equal(t, "mock refresh error", err.Error(), "Error message should match")
}

// StandardOAuthProviderMock is a simplified implementation of StandardOAuthProvider for testing
type StandardOAuthProviderMock struct {
	clientProvider OAuthClientProvider
	serverURL      string
}

// GetToken implements the OAuthProvider interface
func (p *StandardOAuthProviderMock) GetToken() (string, error) {
	tokens, err := p.clientProvider.Tokens()
	if err != nil {
		return "", err
	}

	if tokens == nil {
		return "", errors.New("no tokens available")
	}

	// Simulate token expiration logic
	mockProvider, ok := p.clientProvider.(*MockOAuthClientProvider)
	if ok && mockProvider.shouldRefresh {
		// Directly refresh and return new token
		refreshed, err := mockProvider.refreshToken()
		if err != nil {
			return "", err
		}
		return refreshed.AccessToken, nil
	}

	return tokens.AccessToken, nil
}

// RefreshToken implements the OAuthProvider interface
func (p *StandardOAuthProviderMock) RefreshToken() (string, error) {
	mockProvider, ok := p.clientProvider.(*MockOAuthClientProvider)
	if !ok {
		return "", errors.New("client provider is not a MockOAuthClientProvider")
	}

	// Directly call mock refresh method
	if mockProvider.shouldFailRefresh {
		return "", errors.New("mock refresh error")
	}

	refreshed, err := mockProvider.refreshToken()
	if err != nil {
		return "", err
	}

	return refreshed.AccessToken, nil
}

// MockOAuthClientProvider is a mock implementation of OAuthClientProvider for testing
type MockOAuthClientProvider struct {
	redirectURL       string
	clientMetadata    *common.OAuthClientMetadata
	clientInfo        *common.OAuthClientInformation
	tokens            *common.OAuthTokens
	codeVerifier      string
	shouldRefresh     bool
	shouldFailRefresh bool
}

func (p *MockOAuthClientProvider) RedirectURL() string {
	return p.redirectURL
}

func (p *MockOAuthClientProvider) ClientMetadata() *common.OAuthClientMetadata {
	return p.clientMetadata
}

func (p *MockOAuthClientProvider) ClientInformation() (*common.OAuthClientInformation, error) {
	return p.clientInfo, nil
}

func (p *MockOAuthClientProvider) SaveClientInformation(info *common.OAuthClientInformationFull) error {
	p.clientInfo = &common.OAuthClientInformation{
		ClientID:              info.ClientID,
		ClientSecret:          info.ClientSecret,
		ClientIDIssuedAt:      info.ClientIDIssuedAt,
		ClientSecretExpiresAt: info.ClientSecretExpiresAt,
	}
	return nil
}

func (p *MockOAuthClientProvider) Tokens() (*common.OAuthTokens, error) {
	if p.shouldRefresh {
		// Simulate token expiration
		return &common.OAuthTokens{
			AccessToken:  "expired-token",
			RefreshToken: p.tokens.RefreshToken,
			TokenType:    p.tokens.TokenType,
			ExpiresIn:    0,
		}, nil
	}
	return p.tokens, nil
}

func (p *MockOAuthClientProvider) SaveTokens(tokens *common.OAuthTokens) error {
	p.tokens = tokens
	return nil
}

func (p *MockOAuthClientProvider) RedirectToAuthorization(authURL *url.URL) error {
	return nil
}

func (p *MockOAuthClientProvider) SaveCodeVerifier(codeVerifier string) error {
	p.codeVerifier = codeVerifier
	return nil
}

func (p *MockOAuthClientProvider) CodeVerifier() (string, error) {
	return p.codeVerifier, nil
}

// Helper method for testing, simulates token refresh
func (p *MockOAuthClientProvider) refreshToken() (*common.OAuthTokens, error) {
	if p.shouldFailRefresh {
		return nil, errors.New("mock refresh error")
	}

	// Modify current token instead of returning a new one
	refreshed := &common.OAuthTokens{
		AccessToken:  "refreshed-access-token",
		RefreshToken: "refreshed-refresh-token",
		TokenType:    "bearer",
		ExpiresIn:    3600,
	}
	p.tokens = refreshed
	return refreshed, nil
}
