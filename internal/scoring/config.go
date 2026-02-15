package scoring

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Score scale constants used in breakpoint definitions.
// Using named constants improves readability and reduces magic number count.
const (
	ScoreExcellent = 10.0
	ScoreGood      = 8.0
	ScoreAboveAvg  = 7.0
	ScoreAdequate  = 6.0
	ScoreBelowAvg  = 5.0
	ScorePoor      = 4.0
	ScoreWeak      = 3.0
	ScoreVeryPoor  = 2.0
	ScoreMinimum   = 1.0
)

// PercentPerfect is the breakpoint value for 100% (perfect) percentage metrics.
const PercentPerfect = 100

// Tier threshold constants for composite score classification.
const (
	TierReadyMin    = 8.0
	TierAssistedMin = 6.0
	TierLimitedMin  = 4.0
	TierHostileMin  = 1.0
)

// Category weight constants define the relative importance of each category
// in the composite score. Must sum to ~1.0 across all categories.
const (
	WeightCodeHealth    = 0.25
	WeightSemantics     = 0.10
	WeightArchitecture  = 0.20
	WeightDocumentation = 0.15
	WeightTemporal      = 0.10
	WeightTesting       = 0.15
	WeightAgentEval     = 0.10
)

// Metric weight constants define the relative importance of sub-metrics
// within each category. Reused across categories where weight values coincide.
const (
	MetricWeightPrimary  = 0.30
	MetricWeightHigh     = 0.25
	MetricWeightMedium   = 0.20
	MetricWeightStandard = 0.15
	MetricWeightLow      = 0.10
	MetricWeightMinimal  = 0.05
)

// C1 breakpoint value constants.
const (
	ComplexityGood     = 5.0
	ComplexityAdequate = 10.0
	ComplexityWeak     = 20.0
	ComplexityCritical = 40.0

	FuncLenGood     = 5.0
	FuncLenAdequate = 15.0
	FuncLenWeak     = 30.0
	FuncLenPoor     = 60.0
	FuncLenCritical = 100.0

	FileSizeGood     = 50.0
	FileSizeAdequate = 150.0
	FileSizeWeak     = 300.0
	FileSizePoor     = 500.0
	FileSizeCritical = 1000.0

	CouplingAdequate = 5.0
	CouplingWeak     = 10.0
	CouplingCritical = 20.0

	DupRateGood     = 3.0
	DupRateAdequate = 8.0
	DupRateWeak     = 15.0
	DupRateCritical = 50.0
)

// C2 breakpoint value constants.
const (
	TypeAnnotationWeak     = 30.0
	TypeAnnotationAdequate = 50.0
	TypeAnnotationGood     = 80.0

	NamingWeak     = 70.0
	NamingAdequate = 85.0
	NamingGood     = 95.0

	MagicNumGood     = 5.0
	MagicNumAdequate = 15.0
	MagicNumWeak     = 30.0
	MagicNumCritical = 50.0

	NullSafetyWeak     = 30.0
	NullSafetyAdequate = 50.0
	NullSafetyGood     = 80.0
)

// C3 breakpoint value constants.
const (
	DirDepthGood     = 3.0
	DirDepthAdequate = 5.0
	DirDepthWeak     = 7.0
	DirDepthCritical = 10.0

	FanoutGood     = 3.0
	FanoutAdequate = 6.0
	FanoutWeak     = 10.0
	FanoutCritical = 15.0

	CircDepsAdequate = 3.0
	CircDepsPoor     = 5.0
	CircDepsCritical = 10.0

	ImportCompGood     = 4.0
	ImportCompAdequate = 6.0
	ImportCompCritical = 8.0

	DeadExportsGood     = 5.0
	DeadExportsAdequate = 15.0
	DeadExportsWeak     = 30.0
	DeadExportsCritical = 50.0
)

// C4 breakpoint value constants.
const (
	ReadmeWordsWeak     = 100.0
	ReadmeWordsAdequate = 300.0
	ReadmeWordsGood     = 500.0
	ReadmeWordsExcel    = 1000.0

	CommentDensityWeak     = 5.0
	CommentDensityAdequate = 10.0
	CommentDensityGood     = 15.0
	CommentDensityExcel    = 25.0

	APIDocWeak     = 30.0
	APIDocAdequate = 50.0
	APIDocGood     = 80.0
)

// C5 breakpoint value constants.
const (
	ChurnExcel    = 50.0
	ChurnGood     = 100.0
	ChurnAdequate = 300.0
	ChurnWeak     = 600.0
	ChurnCritical = 1000.0

	TempCouplingGood     = 5.0
	TempCouplingAdequate = 15.0
	TempCouplingWeak     = 25.0
	TempCouplingCritical = 30.0

	AuthorFragGood     = 4.0
	AuthorFragWeak     = 6.0
	AuthorFragCritical = 8.0

	CommitStabMinimum  = 0.5
	CommitStabAdequate = 3.0
	CommitStabGood     = 7.0
	CommitStabExcel    = 14.0

	HotspotExcel    = 20.0
	HotspotGood     = 30.0
	HotspotAdequate = 50.0
	HotspotWeak     = 70.0
	HotspotCritical = 80.0
)

// C6 breakpoint value constants.
const (
	TestRatioPoor     = 0.2
	TestRatioAdequate = 0.5
	TestRatioGood     = 0.8
	TestRatioExcel    = 1.5

	CoveragePoor     = 30.0
	CoverageAdequate = 50.0
	CoverageGood     = 70.0
	CoverageExcel    = 90.0

	IsolationPoor     = 40.0
	IsolationAdequate = 60.0
	IsolationGood     = 80.0
	IsolationExcel    = 95.0

	AssertDensityAdequate = 3.0
	AssertDensityExcel    = 5.0

	TestFileRatioPoor     = 0.3
	TestFileRatioAdequate = 0.5
	TestFileRatioGood     = 0.7
	TestFileRatioExcel    = 0.9
)

// C7 breakpoint value constants.
const (
	C7ScorePoor    = 4.0
	C7ScoreAboveAvg = 7.0
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

// tierConfig defines a tier rating boundary.
// Tiers should be sorted by MinScore descending.
type tierConfig struct {
	Name     string  `yaml:"name"`
	MinScore float64 `yaml:"min_score"`
}

// ScoringConfig holds all scoring thresholds and weights.
// Categories are stored in a map keyed by category identifier (e.g., "C1", "C2").
type ScoringConfig struct {
	Categories map[string]CategoryConfig `yaml:"categories"`
	Tiers      []tierConfig             `yaml:"tiers"`
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
			"C1": defaultC1Config(),
			"C2": defaultC2Config(),
			"C3": defaultC3Config(),
			"C4": defaultC4Config(),
			"C5": defaultC5Config(),
			"C6": defaultC6Config(),
			"C7": defaultC7Config(),
		},
		Tiers: []tierConfig{
			{Name: "Agent-Ready", MinScore: TierReadyMin},
			{Name: "Agent-Assisted", MinScore: TierAssistedMin},
			{Name: "Agent-Limited", MinScore: TierLimitedMin},
			{Name: "Agent-Hostile", MinScore: TierHostileMin},
		},
	}
}

func defaultC1Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Code Health",
		Weight: WeightCodeHealth,
		Metrics: []MetricThresholds{
			{Name: "complexity_avg", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreExcellent}, {Value: ComplexityGood, Score: ScoreGood},
				{Value: ComplexityAdequate, Score: ScoreAdequate}, {Value: ComplexityWeak, Score: ScoreWeak},
				{Value: ComplexityCritical, Score: ScoreMinimum},
			}},
			{Name: "func_length_avg", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: FuncLenGood, Score: ScoreExcellent}, {Value: FuncLenAdequate, Score: ScoreGood},
				{Value: FuncLenWeak, Score: ScoreAdequate}, {Value: FuncLenPoor, Score: ScoreWeak},
				{Value: FuncLenCritical, Score: ScoreMinimum},
			}},
			{Name: "file_size_avg", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: FileSizeGood, Score: ScoreExcellent}, {Value: FileSizeAdequate, Score: ScoreGood},
				{Value: FileSizeWeak, Score: ScoreAdequate}, {Value: FileSizePoor, Score: ScoreWeak},
				{Value: FileSizeCritical, Score: ScoreMinimum},
			}},
			{Name: "afferent_coupling_avg", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: 2, Score: ScoreGood},
				{Value: CouplingAdequate, Score: ScoreAdequate}, {Value: CouplingWeak, Score: ScoreWeak},
				{Value: CouplingCritical, Score: ScoreMinimum},
			}},
			{Name: "efferent_coupling_avg", Weight: MetricWeightLow, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: 2, Score: ScoreGood},
				{Value: CouplingAdequate, Score: ScoreAdequate}, {Value: CouplingWeak, Score: ScoreWeak},
				{Value: CouplingCritical, Score: ScoreMinimum},
			}},
			{Name: "duplication_rate", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: DupRateGood, Score: ScoreGood},
				{Value: DupRateAdequate, Score: ScoreAdequate}, {Value: DupRateWeak, Score: ScoreWeak},
				{Value: DupRateCritical, Score: ScoreMinimum},
			}},
		},
	}
}

func defaultC2Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Semantic Explicitness",
		Weight: WeightSemantics,
		Metrics: []MetricThresholds{
			{Name: "type_annotation_coverage", Weight: MetricWeightPrimary, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: TypeAnnotationWeak, Score: ScoreWeak},
				{Value: TypeAnnotationAdequate, Score: ScoreAdequate}, {Value: TypeAnnotationGood, Score: ScoreGood},
				{Value: PercentPerfect, Score: ScoreExcellent},
			}},
			{Name: "naming_consistency", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: NamingWeak, Score: ScoreWeak},
				{Value: NamingAdequate, Score: ScoreAdequate}, {Value: NamingGood, Score: ScoreGood},
				{Value: PercentPerfect, Score: ScoreExcellent},
			}},
			{Name: "magic_number_ratio", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: MagicNumGood, Score: ScoreGood},
				{Value: MagicNumAdequate, Score: ScoreAdequate}, {Value: MagicNumWeak, Score: ScoreWeak},
				{Value: MagicNumCritical, Score: ScoreMinimum},
			}},
			{Name: "type_strictness", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreWeak}, {Value: 1, Score: ScoreExcellent},
			}},
			{Name: "null_safety", Weight: MetricWeightLow, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: NullSafetyWeak, Score: ScoreWeak},
				{Value: NullSafetyAdequate, Score: ScoreAdequate}, {Value: NullSafetyGood, Score: ScoreGood},
				{Value: PercentPerfect, Score: ScoreExcellent},
			}},
		},
	}
}

func defaultC3Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Architecture",
		Weight: WeightArchitecture,
		Metrics: []MetricThresholds{
			{Name: "max_dir_depth", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreExcellent}, {Value: DirDepthGood, Score: ScoreGood},
				{Value: DirDepthAdequate, Score: ScoreAdequate}, {Value: DirDepthWeak, Score: ScoreWeak},
				{Value: DirDepthCritical, Score: ScoreMinimum},
			}},
			{Name: "module_fanout_avg", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: FanoutGood, Score: ScoreGood},
				{Value: FanoutAdequate, Score: ScoreAdequate}, {Value: FanoutWeak, Score: ScoreWeak},
				{Value: FanoutCritical, Score: ScoreMinimum},
			}},
			{Name: "circular_deps", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: 1, Score: ScoreAdequate},
				{Value: CircDepsAdequate, Score: ScoreWeak}, {Value: CircDepsPoor, Score: ScoreVeryPoor},
				{Value: CircDepsCritical, Score: ScoreMinimum},
			}},
			{Name: "import_complexity_avg", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreExcellent}, {Value: 2, Score: ScoreGood},
				{Value: ImportCompGood, Score: ScoreAdequate}, {Value: ImportCompAdequate, Score: ScoreWeak},
				{Value: ImportCompCritical, Score: ScoreMinimum},
			}},
			{Name: "dead_exports", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: DeadExportsGood, Score: ScoreGood},
				{Value: DeadExportsAdequate, Score: ScoreAdequate}, {Value: DeadExportsWeak, Score: ScoreWeak},
				{Value: DeadExportsCritical, Score: ScoreMinimum},
			}},
		},
	}
}

func defaultC4Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Documentation Quality",
		Weight: WeightDocumentation,
		Metrics: []MetricThresholds{
			{Name: "readme_word_count", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: ReadmeWordsWeak, Score: ScoreWeak},
				{Value: ReadmeWordsAdequate, Score: ScoreAdequate}, {Value: ReadmeWordsGood, Score: ScoreGood},
				{Value: ReadmeWordsExcel, Score: ScoreExcellent},
			}},
			{Name: "comment_density", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: CommentDensityWeak, Score: ScoreWeak},
				{Value: CommentDensityAdequate, Score: ScoreAdequate}, {Value: CommentDensityGood, Score: ScoreGood},
				{Value: CommentDensityExcel, Score: ScoreExcellent},
			}},
			{Name: "api_doc_coverage", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: APIDocWeak, Score: ScoreWeak},
				{Value: APIDocAdequate, Score: ScoreAdequate}, {Value: APIDocGood, Score: ScoreGood},
				{Value: PercentPerfect, Score: ScoreExcellent},
			}},
			{Name: "changelog_present", Weight: MetricWeightLow, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreWeak}, {Value: 1, Score: ScoreExcellent},
			}},
			{Name: "examples_present", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreWeak}, {Value: 1, Score: ScoreExcellent},
			}},
			{Name: "contributing_present", Weight: MetricWeightLow, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreWeak}, {Value: 1, Score: ScoreExcellent},
			}},
			{Name: "diagrams_present", Weight: MetricWeightMinimal, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreBelowAvg}, {Value: 1, Score: ScoreExcellent},
			}},
		},
	}
}

func defaultC5Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Temporal Dynamics",
		Weight: WeightTemporal,
		Metrics: []MetricThresholds{
			{Name: "churn_rate", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: ChurnExcel, Score: ScoreExcellent}, {Value: ChurnGood, Score: ScoreGood},
				{Value: ChurnAdequate, Score: ScoreAdequate}, {Value: ChurnWeak, Score: ScoreWeak},
				{Value: ChurnCritical, Score: ScoreMinimum},
			}},
			{Name: "temporal_coupling_pct", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreExcellent}, {Value: TempCouplingGood, Score: ScoreGood},
				{Value: TempCouplingAdequate, Score: ScoreAdequate}, {Value: TempCouplingWeak, Score: ScoreWeak},
				{Value: TempCouplingCritical, Score: ScoreMinimum},
			}},
			{Name: "author_fragmentation", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreExcellent}, {Value: 2, Score: ScoreGood},
				{Value: AuthorFragGood, Score: ScoreAdequate}, {Value: AuthorFragWeak, Score: ScoreWeak},
				{Value: AuthorFragCritical, Score: ScoreMinimum},
			}},
			{Name: "commit_stability", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: CommitStabMinimum, Score: ScoreMinimum}, {Value: 1, Score: ScoreWeak},
				{Value: CommitStabAdequate, Score: ScoreAdequate}, {Value: CommitStabGood, Score: ScoreGood},
				{Value: CommitStabExcel, Score: ScoreExcellent},
			}},
			{Name: "hotspot_concentration", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: HotspotExcel, Score: ScoreExcellent}, {Value: HotspotGood, Score: ScoreGood},
				{Value: HotspotAdequate, Score: ScoreAdequate}, {Value: HotspotWeak, Score: ScoreWeak},
				{Value: HotspotCritical, Score: ScoreMinimum},
			}},
		},
	}
}

func defaultC6Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Testing",
		Weight: WeightTesting,
		Metrics: []MetricThresholds{
			{Name: "test_to_code_ratio", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: TestRatioPoor, Score: ScorePoor},
				{Value: TestRatioAdequate, Score: ScoreAdequate}, {Value: TestRatioGood, Score: ScoreGood},
				{Value: TestRatioExcel, Score: ScoreExcellent},
			}},
			{Name: "coverage_percent", Weight: MetricWeightPrimary, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: CoveragePoor, Score: ScorePoor},
				{Value: CoverageAdequate, Score: ScoreAdequate}, {Value: CoverageGood, Score: ScoreGood},
				{Value: CoverageExcel, Score: ScoreExcellent},
			}},
			{Name: "test_isolation", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: IsolationPoor, Score: ScorePoor},
				{Value: IsolationAdequate, Score: ScoreAdequate}, {Value: IsolationGood, Score: ScoreGood},
				{Value: IsolationExcel, Score: ScoreExcellent},
			}},
			{Name: "assertion_density_avg", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: 1, Score: ScorePoor},
				{Value: 2, Score: ScoreAdequate}, {Value: AssertDensityAdequate, Score: ScoreGood},
				{Value: AssertDensityExcel, Score: ScoreExcellent},
			}},
			{Name: "test_file_ratio", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 0, Score: ScoreMinimum}, {Value: TestFileRatioPoor, Score: ScorePoor},
				{Value: TestFileRatioAdequate, Score: ScoreAdequate}, {Value: TestFileRatioGood, Score: ScoreGood},
				{Value: TestFileRatioExcel, Score: ScoreExcellent},
			}},
		},
	}
}

func defaultC7Config() CategoryConfig {
	return CategoryConfig{
		Name:   "Agent Evaluation",
		Weight: WeightAgentEval,
		Metrics: []MetricThresholds{
			{Name: "task_execution_consistency", Weight: MetricWeightMedium, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreMinimum}, {Value: C7ScorePoor, Score: ScorePoor},
				{Value: C7ScoreAboveAvg, Score: ScoreAboveAvg}, {Value: 10, Score: ScoreExcellent},
			}},
			{Name: "code_behavior_comprehension", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreMinimum}, {Value: C7ScorePoor, Score: ScorePoor},
				{Value: C7ScoreAboveAvg, Score: ScoreAboveAvg}, {Value: 10, Score: ScoreExcellent},
			}},
			{Name: "cross_file_navigation", Weight: MetricWeightHigh, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreMinimum}, {Value: C7ScorePoor, Score: ScorePoor},
				{Value: C7ScoreAboveAvg, Score: ScoreAboveAvg}, {Value: 10, Score: ScoreExcellent},
			}},
			{Name: "identifier_interpretability", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreMinimum}, {Value: C7ScorePoor, Score: ScorePoor},
				{Value: C7ScoreAboveAvg, Score: ScoreAboveAvg}, {Value: 10, Score: ScoreExcellent},
			}},
			{Name: "documentation_accuracy_detection", Weight: MetricWeightStandard, Breakpoints: []Breakpoint{
				{Value: 1, Score: ScoreMinimum}, {Value: C7ScorePoor, Score: ScorePoor},
				{Value: C7ScoreAboveAvg, Score: ScoreAboveAvg}, {Value: 10, Score: ScoreExcellent},
			}},
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
