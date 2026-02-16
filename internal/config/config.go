package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	DataDir  string         `yaml:"data_dir"`
	Auth     AuthConfig     `yaml:"auth"`
	Agent    AgentConfig    `yaml:"agent"`
	Cloud    CloudConfig    `yaml:"cloud"`
	Branding BrandingConfig `yaml:"branding"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	SecretKey string `yaml:"secret_key"`
	BaseURL   string `yaml:"base_url"` // e.g. https://aetherdev.example.com — used for OIDC redirect URI
}

// AuthConfig configures authentication. When OIDC is enabled, users log in via
// their company's identity provider. When disabled, the v1 default admin is used.
type AuthConfig struct {
	OIDCEnabled  bool   `yaml:"oidc_enabled"`
	IssuerURL    string `yaml:"issuer_url"`     // e.g. https://accounts.google.com or https://login.microsoftonline.com/{tenant}/v2.0
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	// Scopes defaults to "openid profile email" if empty.
	Scopes []string `yaml:"scopes"`
}

type AgentConfig struct {
	ClaudeAPIKey     string `yaml:"claude_api_key"`
	ClaudeModel      string `yaml:"claude_model"`
	AWSRegion        string `yaml:"aws_region"`
	AWSAgentID       string `yaml:"aws_agent_id"`
	AWSAgentAliasID  string `yaml:"aws_agent_alias_id"`
	EnableAutoReview bool   `yaml:"enable_auto_review"`
	EnableAutoFix    bool   `yaml:"enable_auto_fix"`
}

type CloudConfig struct {
	Provider         string `yaml:"provider"` // aws, gcp, azure
	DefaultRegion    string `yaml:"default_region"`
	IaCFramework     string `yaml:"iac_framework"` // terraform, pulumi, cdk
	EnableCostGuard  bool   `yaml:"enable_cost_guard"`
	EnableCompliance bool   `yaml:"enable_compliance"`
}

type BrandingConfig struct {
	AppName  string `yaml:"app_name"`
	LogoPath string `yaml:"logo_path"`
	Theme    string `yaml:"theme"` // dark, light, auto
	Accent   string `yaml:"accent"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:      3000,
			Host:      "0.0.0.0",
			SecretKey: "change-me-in-production",
		},
		DataDir: "./data",
		Agent: AgentConfig{
			ClaudeModel:      "claude-sonnet-4-20250514",
			AWSRegion:        "us-east-1",
			EnableAutoReview: true,
			EnableAutoFix:    false,
		},
		Cloud: CloudConfig{
			Provider:         "aws",
			DefaultRegion:    "us-east-1",
			IaCFramework:     "terraform",
			EnableCostGuard:  true,
			EnableCompliance: true,
		},
		Branding: BrandingConfig{
			AppName:  "AetherDev",
			LogoPath: "/static/img/logo.svg",
			Theme:    "dark",
			Accent:   "#6C5CE7",
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) RepoRootPath() string {
	return filepath.Join(c.DataDir, "repositories")
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "aetherdev.db")
}

// OIDCRedirectURI returns the callback URL the IdP should redirect to.
func (c *Config) OIDCRedirectURI() string {
	base := c.Server.BaseURL
	if base == "" {
		base = fmt.Sprintf("http://localhost:%d", c.Server.Port)
	}
	return base + "/auth/callback"
}

// OIDCScopes returns the scopes to request, defaulting to "openid profile email".
func (c *Config) OIDCScopes() []string {
	if len(c.Auth.Scopes) > 0 {
		return c.Auth.Scopes
	}
	return []string{"openid", "profile", "email"}
}
