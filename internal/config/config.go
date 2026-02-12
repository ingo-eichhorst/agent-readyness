// Package config handles .arsrc.yml project-level configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/ingo/agent-readyness/internal/scoring"
)

// ProjectConfig represents the .arsrc.yml configuration file.
type ProjectConfig struct {
	Version   int               `yaml:"version"`
	Scoring   scoringOverrides  `yaml:"scoring"`
	Languages []string          `yaml:"languages"`
	Metrics   map[string]metricOverrides `yaml:"metrics"`
}

// scoringOverrides contains weight and threshold overrides.
type scoringOverrides struct {
	Weights   map[string]float64 `yaml:"weights"`
	Threshold float64            `yaml:"threshold"`
}

// metricOverrides allows per-metric customization.
type metricOverrides struct {
	Enabled   *bool   `yaml:"enabled"`
	Threshold float64 `yaml:"threshold"`
}

// LoadProjectConfig loads project configuration from .arsrc.yml or .arsrc.yaml.
// If explicitPath is provided (from --config flag), that file is loaded.
// Otherwise, looks for .arsrc.yml then .arsrc.yaml in dir.
// Returns nil (no error) if no config file is found.
func LoadProjectConfig(dir string, explicitPath string) (*ProjectConfig, error) {
	var configPath string

	if explicitPath != "" {
		configPath = explicitPath
	} else {
		// Look for .arsrc.yml then .arsrc.yaml
		ymlPath := filepath.Join(dir, ".arsrc.yml")
		yamlPath := filepath.Join(dir, ".arsrc.yaml")

		if _, err := os.Stat(ymlPath); err == nil {
			configPath = ymlPath
		} else if _, err := os.Stat(yamlPath); err == nil {
			configPath = yamlPath
		} else {
			return nil, nil // No config found, use defaults
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read project config %s: %w", configPath, err)
	}

	cfg := &ProjectConfig{}
	// Use strict decoding to reject unknown fields
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse project config %s: %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid project config %s: %w", configPath, err)
	}

	return cfg, nil
}

// Validate checks that the ProjectConfig values are valid.
func (c *ProjectConfig) Validate() error {
	if c.Version != 0 && c.Version != 1 {
		return fmt.Errorf("unsupported config version %d (expected 1)", c.Version)
	}

	// Validate weights are non-negative
	for name, weight := range c.Scoring.Weights {
		if weight < 0 {
			return fmt.Errorf("weight for %q must be >= 0, got %f", name, weight)
		}
	}

	// Validate threshold is non-negative
	if c.Scoring.Threshold < 0 {
		return fmt.Errorf("threshold must be >= 0, got %f", c.Scoring.Threshold)
	}

	return nil
}

// ApplyToScoringConfig applies project config overrides to a ScoringConfig.
func (c *ProjectConfig) ApplyToScoringConfig(sc *scoring.ScoringConfig) {
	if c == nil || sc == nil {
		return
	}

	// Override category weights
	for catName, weight := range c.Scoring.Weights {
		if cat, ok := sc.Categories[catName]; ok {
			cat.Weight = weight
			sc.Categories[catName] = cat
		}
	}
}
