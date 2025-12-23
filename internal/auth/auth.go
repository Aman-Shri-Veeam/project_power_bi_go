package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/veeam/powerbi-backup-go/internal/config"
	"github.com/veeam/powerbi-backup-go/internal/logger"
)

// AuthService handles authentication with Azure AD
type AuthService struct {
	clientID     string
	clientSecret string
	tenantID     string
	resource     string
	authorityURL string
	tokenCache   string
	mu           sync.RWMutex
}

// TokenResponse represents the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   string `json:"expires_in"`
}

// NewAuthService creates a new authentication service
func NewAuthService(settings *config.Settings) *AuthService {
	return &AuthService{
		clientID:     settings.PowerBIClientID,
		clientSecret: settings.PowerBIClientSecret,
		tenantID:     settings.PowerBITenantID,
		resource:     settings.Resource,
		authorityURL: settings.AuthorityURL,
	}
}

// GetAccessToken retrieves an access token for Power BI API
func (a *AuthService) GetAccessToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	if a.tokenCache != "" {
		token := a.tokenCache
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	tokenURL := fmt.Sprintf("%s/%s/oauth2/token", a.authorityURL, a.tenantID)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)
	data.Set("resource", a.resource)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		logger.LogError("Failed to create token request", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError("Failed to obtain access token", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError("Failed to read token response", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(fmt.Sprintf("Failed to obtain access token: %s", string(body)), nil)
		return "", fmt.Errorf("failed to obtain token: %s", string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		logger.LogError("Failed to parse token response", err)
		return "", err
	}

	a.mu.Lock()
	a.tokenCache = tokenResp.AccessToken
	a.mu.Unlock()

	logger.LogInfo("Access token obtained successfully")
	return tokenResp.AccessToken, nil
}

// ClearTokenCache clears the cached token (useful for testing or token refresh)
func (a *AuthService) ClearTokenCache() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tokenCache = ""
}
