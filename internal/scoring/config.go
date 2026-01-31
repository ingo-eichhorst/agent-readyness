package scoring

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Breakpoint defines a mapping from a raw metric value to a score.
// Breakpoints must be sorted by Value in ascending order.
type Breakpoint struct {
	Value float64 `yaml:"value"` // raw metric value
	Score float64 `yaml:"score"` // corresponding score (1-10)
}

// MetricThresholds defines the breakpoints for scoring a single metric.
type MetricThresholds struct {
	Name        string       `yaml:"name"`
	Weight      float64      `yaml:"weight"`
	Breakpoints []Breakpoint `yaml:"breakpoints"`
}

// CategoryConfig defines the scoring configuration for one category.
type CategoryConfig struct {
	Name    string             `yaml:"name"`
	Weight  float64            `yaml:"weight"`
	Metrics []MetricThresholds `yaml:"metrics"`
}

// TierConfig defines a tier rating boundary.
// Tiers should be sorted by MinScore descending.
type TierConfig struct {
	Name     string  `yaml:"name"`
	MinScore float64 `yaml:"min_score"`
}

// ScoringConfig holds all scoring thresholds and weights.
type ScoringConfig struct {
	C1    CategoryConfig `yaml:"c1"`
	C3    CategoryConfig `yaml:"c3"`
	C6    CategoryConfig `yaml:"c6"`
	Tiers []TierConfig   `yaml:"tiers"`
}

// DefaultConfig returns the default scoring configuration with breakpoints
// for all 16 metrics across C1, C3, and C6 categories.
func DefaultConfig() *ScoringConfig {
	return &ScoringConfig{
		C1: CategoryConfig{
			Name:   "Code Health",
			Weight: 0.25,
			Metrics: []MetricThresholds{
				{
					Name:   "complexity_avg",
					Weight: 0.25,
					Breakpoints: []Breakpoint{
						{Value: 1, Score: 10},
						{Value: 5, Score: 8},
						{Value: 10, Score: 6},
						{Value: 20, Score: 3},
						{Value: 40, Score: 1},
					},
				},
				{
					Name:   "func_length_avg",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 5, Score: 10},
						{Value: 15, Score: 8},
						{Value: 30, Score: 6},
						{Value: 60, Score: 3},
						{Value: 100, Score: 1},
					},
				},
				{
					Name:   "file_size_avg",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 50, Score: 10},
						{Value: 150, Score: 8},
						{Value: 300, Score: 6},
						{Value: 500, Score: 3},
						{Value: 1000, Score: 1},
					},
				},
				{
					Name:   "afferent_coupling_avg",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 2, Score: 8},
						{Value: 5, Score: 6},
						{Value: 10, Score: 3},
						{Value: 20, Score: 1},
					},
				},
				{
					Name:   "efferent_coupling_avg",
					Weight: 0.10,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 2, Score: 8},
						{Value: 5, Score: 6},
						{Value: 10, Score: 3},
						{Value: 20, Score: 1},
					},
				},
				{
					Name:   "duplication_rate",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 3, Score: 8},
						{Value: 8, Score: 6},
						{Value: 15, Score: 3},
						{Value: 50, Score: 1},
					},
				},
			},
		},
		C3: CategoryConfig{
			Name:   "Architecture",
			Weight: 0.20,
			Metrics: []MetricThresholds{
				{
					Name:   "max_dir_depth",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 1, Score: 10},
						{Value: 3, Score: 8},
						{Value: 5, Score: 6},
						{Value: 7, Score: 3},
						{Value: 10, Score: 1},
					},
				},
				{
					Name:   "module_fanout_avg",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 3, Score: 8},
						{Value: 6, Score: 6},
						{Value: 10, Score: 3},
						{Value: 15, Score: 1},
					},
				},
				{
					Name:   "circular_deps",
					Weight: 0.25,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 1, Score: 6},
						{Value: 3, Score: 3},
						{Value: 5, Score: 2},
						{Value: 10, Score: 1},
					},
				},
				{
					Name:   "import_complexity_avg",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 1, Score: 10},
						{Value: 2, Score: 8},
						{Value: 4, Score: 6},
						{Value: 6, Score: 3},
						{Value: 8, Score: 1},
					},
				},
				{
					Name:   "dead_exports",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 5, Score: 8},
						{Value: 15, Score: 6},
						{Value: 30, Score: 3},
						{Value: 50, Score: 1},
					},
				},
			},
		},
		C6: CategoryConfig{
			Name:   "Testing",
			Weight: 0.15,
			Metrics: []MetricThresholds{
				{
					Name:   "test_to_code_ratio",
					Weight: 0.25,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 1},
						{Value: 0.2, Score: 4},
						{Value: 0.5, Score: 6},
						{Value: 0.8, Score: 8},
						{Value: 1.5, Score: 10},
					},
				},
				{
					Name:   "coverage_percent",
					Weight: 0.30,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 1},
						{Value: 30, Score: 4},
						{Value: 50, Score: 6},
						{Value: 70, Score: 8},
						{Value: 90, Score: 10},
					},
				},
				{
					Name:   "test_isolation",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 1},
						{Value: 40, Score: 4},
						{Value: 60, Score: 6},
						{Value: 80, Score: 8},
						{Value: 95, Score: 10},
					},
				},
				{
					Name:   "assertion_density_avg",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 1},
						{Value: 1, Score: 4},
						{Value: 2, Score: 6},
						{Value: 3, Score: 8},
						{Value: 5, Score: 10},
					},
				},
				{
					Name:   "test_file_ratio",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 1},
						{Value: 0.3, Score: 4},
						{Value: 0.5, Score: 6},
						{Value: 0.7, Score: 8},
						{Value: 0.9, Score: 10},
					},
				},
			},
		},
		Tiers: []TierConfig{
			{Name: "Agent-Ready", MinScore: 8.0},
			{Name: "Agent-Assisted", MinScore: 6.0},
			{Name: "Agent-Limited", MinScore: 4.0},
			{Name: "Agent-Hostile", MinScore: 1.0},
		},
	}
}

// LoadConfig loads a ScoringConfig from a YAML file at path.
// If path is empty, returns DefaultConfig().
// The YAML is unmarshaled into a copy of DefaultConfig so that
// missing fields retain their default values.
func LoadConfig(path string) (*ScoringConfig, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scoring config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse scoring config: %w", err)
	}

	return cfg, nil
}
