package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Providers []ProviderConfig `yaml:"providers"`
	Resources ResourceConfig   `yaml:"resources"`
	Export    ExportConfig     `yaml:"export"`
}

// ProviderConfig defines cloud provider configuration
type ProviderConfig struct {
	Name     string                 `yaml:"name"`     // aws, gcp, okta, jfrog, etc.
	Accounts []string               `yaml:"accounts"` // specific accounts to inspect (empty = all)
	Regions  []string               `yaml:"regions"`  // specific regions (empty = all)
	Options  map[string]interface{} `yaml:"options"`  // provider-specific options
}

// ResourceConfig defines which resources to inspect
type ResourceConfig struct {
	Types         []string `yaml:"types"`          // specific resource types (empty = all)
	IncludeAll    bool     `yaml:"include_all"`    // include all resource types
	Relationships bool     `yaml:"relationships"`  // track relationships between resources
}

// ExportConfig defines export settings
type ExportConfig struct {
	Format      string   `yaml:"format"`       // json, yaml, dot, etc.
	OutputFile  string   `yaml:"output_file"`  // output file path
	Pretty      bool     `yaml:"pretty"`       // pretty print output
	IncludeRaw  bool     `yaml:"include_raw"`  // include raw cloud provider data
	Formats     []string `yaml:"formats"`      // multiple output formats
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Resources.Types == nil || len(config.Resources.Types) == 0 {
		config.Resources.IncludeAll = true
	}

	if config.Export.Format == "" && len(config.Export.Formats) == 0 {
		config.Export.Format = "json"
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider must be configured")
	}

	for _, provider := range c.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider name cannot be empty")
		}
	}

	return nil
}
