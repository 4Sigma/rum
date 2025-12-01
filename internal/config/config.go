package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = "rum.yaml"

var (
	ErrConfigNotFound = errors.New("rum.yaml not found")
	ErrConfigParse    = errors.New("failed to parse rum.yaml")
)

// Config is the root configuration structure for rum.yaml.
// It's designed to be extensible for future components.
type Config struct {
	Templates *TemplatesConfig `yaml:"templates,omitempty"`
}

// TemplatesConfig holds configuration for template generation.
type TemplatesConfig struct {
	OutputFile    string           `yaml:"output_file"`
	OutputPackage string           `yaml:"output_package"`
	Sources       []TemplateSource `yaml:"sources"`
}

// TemplateSource defines a template source directory and pattern.
type TemplateSource struct {
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern"`
}

// Load reads and parses the rum.yaml configuration file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Join(ErrConfigParse, err)
	}

	return &cfg, nil
}

// HasTemplates returns true if templates configuration is present.
func (c *Config) HasTemplates() bool {
	return c.Templates != nil && len(c.Templates.Sources) > 0
}
