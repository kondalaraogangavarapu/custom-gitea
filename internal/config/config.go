package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	DataDir  string         `yaml:"data_dir"`
	Agent    AgentConfig    `yaml:"agent"`
	Cloud    CloudConfig    `yaml:"cloud"`
	Branding BrandingConfig `yaml:"branding"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	SecretKey string `yaml:"secret_key"`
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
