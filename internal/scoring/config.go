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
// Categories are stored in a map keyed by category identifier (e.g., "C1", "C2").
type ScoringConfig struct {
	Categories map[string]CategoryConfig `yaml:"categories"`
	Tiers      []TierConfig             `yaml:"tiers"`
}

// Category returns the CategoryConfig for the given category name.
// Returns a zero-value CategoryConfig if the category is not found.
func (sc *ScoringConfig) Category(name string) CategoryConfig {
	if sc.Categories == nil {
		return CategoryConfig{}
	}
	return sc.Categories[name]
}

// DefaultConfig returns the default scoring configuration with breakpoints
// for all metrics across C1, C2, C3, and C6 categories.
func DefaultConfig() *ScoringConfig {
	return &ScoringConfig{
		Categories: map[string]CategoryConfig{
			"C1": {
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
			"C2": {
				Name:   "Semantic Explicitness",
				Weight: 0.10,
				Metrics: []MetricThresholds{
					{
						Name:   "type_annotation_coverage",
						Weight: 0.30,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 30, Score: 3},
							{Value: 50, Score: 6},
							{Value: 80, Score: 8},
							{Value: 100, Score: 10},
						},
					},
					{
						Name:   "naming_consistency",
						Weight: 0.25,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 70, Score: 3},
							{Value: 85, Score: 6},
							{Value: 95, Score: 8},
							{Value: 100, Score: 10},
						},
					},
					{
						Name:   "magic_number_ratio",
						Weight: 0.20,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 10},
							{Value: 5, Score: 8},
							{Value: 15, Score: 6},
							{Value: 30, Score: 3},
							{Value: 50, Score: 1},
						},
					},
					{
						Name:   "type_strictness",
						Weight: 0.15,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 3},
							{Value: 1, Score: 10},
						},
					},
					{
						Name:   "null_safety",
						Weight: 0.10,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 30, Score: 3},
							{Value: 50, Score: 6},
							{Value: 80, Score: 8},
							{Value: 100, Score: 10},
						},
					},
				},
			},
			"C3": {
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
			"C4": {
				Name:   "Documentation Quality",
				Weight: 0.15,
				Metrics: []MetricThresholds{
					{
						Name:   "readme_word_count",
						Weight: 0.15,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 100, Score: 3},
							{Value: 300, Score: 6},
							{Value: 500, Score: 8},
							{Value: 1000, Score: 10},
						},
					},
					{
						Name:   "comment_density",
						Weight: 0.20,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 5, Score: 3},
							{Value: 10, Score: 6},
							{Value: 15, Score: 8},
							{Value: 25, Score: 10},
						},
					},
					{
						Name:   "api_doc_coverage",
						Weight: 0.25,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 30, Score: 3},
							{Value: 50, Score: 6},
							{Value: 80, Score: 8},
							{Value: 100, Score: 10},
						},
					},
					{
						Name:   "changelog_present",
						Weight: 0.10,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 3},
							{Value: 1, Score: 10},
						},
					},
					{
						Name:   "examples_present",
						Weight: 0.15,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 3},
							{Value: 1, Score: 10},
						},
					},
					{
						Name:   "contributing_present",
						Weight: 0.10,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 3},
							{Value: 1, Score: 10},
						},
					},
					{
						Name:   "diagrams_present",
						Weight: 0.05,
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 5},
							{Value: 1, Score: 10},
						},
					},
				},
			},
			"C5": {
			Name:   "Temporal Dynamics",
			Weight: 0.10,
			Metrics: []MetricThresholds{
				{
					Name:   "churn_rate",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 50, Score: 10},
						{Value: 100, Score: 8},
						{Value: 300, Score: 6},
						{Value: 600, Score: 3},
						{Value: 1000, Score: 1},
					},
				},
				{
					Name:   "temporal_coupling_pct",
					Weight: 0.25,
					Breakpoints: []Breakpoint{
						{Value: 0, Score: 10},
						{Value: 5, Score: 8},
						{Value: 15, Score: 6},
						{Value: 25, Score: 3},
						{Value: 30, Score: 1},
					},
				},
				{
					Name:   "author_fragmentation",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 1, Score: 10},
						{Value: 2, Score: 8},
						{Value: 4, Score: 6},
						{Value: 6, Score: 3},
						{Value: 8, Score: 1},
					},
				},
				{
					Name:   "commit_stability",
					Weight: 0.15,
					Breakpoints: []Breakpoint{
						{Value: 0.5, Score: 1},
						{Value: 1, Score: 3},
						{Value: 3, Score: 6},
						{Value: 7, Score: 8},
						{Value: 14, Score: 10},
					},
				},
				{
					Name:   "hotspot_concentration",
					Weight: 0.20,
					Breakpoints: []Breakpoint{
						{Value: 20, Score: 10},
						{Value: 30, Score: 8},
						{Value: 50, Score: 6},
						{Value: 70, Score: 3},
						{Value: 80, Score: 1},
					},
				},
			},
		},
		"C6": {
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
			"C7": {
				Name:   "Agent Evaluation",
				Weight: 0.10,
				Metrics: []MetricThresholds{
					// Legacy overall_score preserved for backward compatibility
					{
						Name:   "overall_score",
						Weight: 0.0, // Zero weight - not used in new scoring
						Breakpoints: []Breakpoint{
							{Value: 0, Score: 1},
							{Value: 30, Score: 3},
							{Value: 50, Score: 5},
							{Value: 70, Score: 7},
							{Value: 90, Score: 10},
						},
					},
					// M1: Task Execution Consistency
					// Measures reproducibility across runs
					// Research: Agent benchmarks show ~13% variance is typical
					{
						Name:   "task_execution_consistency",
						Weight: 0.20,
						Breakpoints: []Breakpoint{
							{Value: 1, Score: 1},   // >30% variance
							{Value: 4, Score: 4},   // 15-30% variance
							{Value: 7, Score: 7},   // 5-15% variance
							{Value: 10, Score: 10}, // <5% variance
						},
					},
					// M2: Code Behavior Comprehension
					// Measures understanding of code semantics (not syntax)
					// Research: LLMs struggle with semantic vs syntactic understanding
					{
						Name:   "code_behavior_comprehension",
						Weight: 0.25,
						Breakpoints: []Breakpoint{
							{Value: 1, Score: 1},   // Fundamentally wrong
							{Value: 4, Score: 4},   // Partial understanding
							{Value: 7, Score: 7},   // Correct main path
							{Value: 10, Score: 10}, // All paths including edge cases
						},
					},
					// M3: Cross-File Navigation
					// Measures dependency tracing across files
					// Research: RepoGraph shows 32.8% improvement with repo-level understanding
					{
						Name:   "cross_file_navigation",
						Weight: 0.25,
						Breakpoints: []Breakpoint{
							{Value: 1, Score: 1},   // Single file only
							{Value: 4, Score: 4},   // Direct dependencies
							{Value: 7, Score: 7},   // Most of chain
							{Value: 10, Score: 10}, // Complete trace
						},
					},
					// M4: Identifier Interpretability
					// Measures ability to infer meaning from names
					// Research: Descriptive compound identifiers improve comprehension
					{
						Name:   "identifier_interpretability",
						Weight: 0.15,
						Breakpoints: []Breakpoint{
							{Value: 1, Score: 1},   // Misinterprets
							{Value: 4, Score: 4},   // Needs context
							{Value: 7, Score: 7},   // Mostly correct
							{Value: 10, Score: 10}, // Correct interpretation
						},
					},
					// M5: Documentation Accuracy Detection
					// Measures ability to detect comment/code mismatches
					// Research: CCI detection is a distinct, measurable capability
					{
						Name:   "documentation_accuracy_detection",
						Weight: 0.15,
						Breakpoints: []Breakpoint{
							{Value: 1, Score: 1},   // Cannot detect
							{Value: 4, Score: 4},   // Obvious only
							{Value: 7, Score: 7},   // Most mismatches
							{Value: 10, Score: 10}, // All mismatches
						},
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
