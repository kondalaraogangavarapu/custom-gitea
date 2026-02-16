// Package oidc implements OpenID Connect discovery and the authorization code
// flow using only the Go standard library. Enterprise admins supply their
// identity provider's issuer URL (which hosts .well-known/openid-configuration)
// and AetherDev handles the rest.
package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProviderConfig is the subset of the OpenID Connect discovery document that
// AetherDev needs.
type ProviderConfig struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// TokenResponse is the response from the token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

// UserInfo is the subset of claims AetherDev reads from the userinfo endpoint.
type UserInfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername  string `json:"preferred_username"`
	Picture           string `json:"picture"`
}

// Client is a configured OIDC relying party.
type Client struct {
	Provider     ProviderConfig
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// Discover fetches the .well-known/openid-configuration from the issuer URL.
func Discover(issuerURL string) (*ProviderConfig, error) {
	wellKnown := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(wellKnown)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var cfg ProviderConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode discovery document: %w", err)
	}
	return &cfg, nil
}

// NewClient creates an OIDC client by discovering the provider config.
func NewClient(issuerURL, clientID, clientSecret, redirectURI string, scopes []string) (*Client, error) {
	provider, err := Discover(issuerURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Provider:     *provider,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       scopes,
	}, nil
}

// AuthURL builds the authorization URL the browser should be redirected to.
// The state parameter should be stored server-side and verified on callback.
func (c *Client) AuthURL(state string) string {
	v := url.Values{
		"response_type": {"code"},
		"client_id":     {c.ClientID},
		"redirect_uri":  {c.RedirectURI},
		"scope":         {strings.Join(c.Scopes, " ")},
		"state":         {state},
	}
	return c.Provider.AuthorizationEndpoint + "?" + v.Encode()
}

// Exchange trades an authorization code for tokens.
func (c *Client) Exchange(code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.RedirectURI},
		"client_id":    {c.ClientID},
		"client_secret": {c.ClientSecret},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(c.Provider.TokenEndpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &tok, nil
}

// FetchUserInfo calls the userinfo endpoint with the access token.
func (c *Client) FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.Provider.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &info, nil
}

// LogoutURL returns the IdP's end-session URL if available.
func (c *Client) LogoutURL(postLogoutRedirect string) string {
	if c.Provider.EndSessionEndpoint == "" {
		return postLogoutRedirect
	}
	v := url.Values{
		"client_id":                {c.ClientID},
		"post_logout_redirect_uri": {postLogoutRedirect},
	}
	return c.Provider.EndSessionEndpoint + "?" + v.Encode()
}

// GenerateState returns a cryptographically random state string for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateSessionID returns a cryptographically random session ID.
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
