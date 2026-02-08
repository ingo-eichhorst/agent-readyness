package types

// Language identifies the programming language of source files.
type Language string

const (
	LangGo         Language = "go"
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
)

// AnalysisTarget is the language-agnostic unit of analysis.
// Each target represents one language found in the project.
type AnalysisTarget struct {
	Language Language
	RootDir  string       // Project root directory
	Files    []SourceFile // Source files for this language
}

// SourceFile represents a single source file for analysis.
type SourceFile struct {
	Path     string
	RelPath  string
	Language Language
	Lines    int
	Content  []byte    // Raw source content (needed for Tree-sitter)
	Class    FileClass // source, test, generated, excluded
}

// FileClass categorizes discovered files in a Go project.
type FileClass int

const (
	ClassSource    FileClass = iota // Regular Go source file
	ClassTest                       // Test file (*_test.go)
	ClassGenerated                  // Generated file (has code generation comment)
	ClassExcluded                   // Excluded file (vendor, .gitignore, etc.)
)

// String returns the human-readable name for a FileClass.
func (fc FileClass) String() string {
	switch fc {
	case ClassSource:
		return "source"
	case ClassTest:
		return "test"
	case ClassGenerated:
		return "generated"
	case ClassExcluded:
		return "excluded"
	default:
		return "unknown"
	}
}

// DiscoveredFile represents a file found during directory scanning.
type DiscoveredFile struct {
	Path          string    // Absolute path to the file
	RelPath       string    // Path relative to project root
	Class         FileClass // Classification of the file
	ExcludeReason string    // Why file was excluded (empty if not excluded)
	Language      Language  // Programming language of the file
}

// ScanResult holds the output of the file discovery phase.
type ScanResult struct {
	RootDir        string              // Absolute path to project root
	TotalFiles     int                 // Total files discovered
	SourceCount    int                 // Number of source files
	TestCount      int                 // Number of test files
	GeneratedCount int                 // Number of generated files
	VendorCount    int                 // Number of vendor-excluded files
	GitignoreCount int                 // Number of gitignore-excluded files
	SkippedCount   int                 // Files/dirs skipped due to errors (permission denied, broken paths, etc.)
	SymlinkCount   int                 // Symlinks detected and skipped
	Files          []DiscoveredFile    // All discovered files
	PerLanguage    map[Language]int    // Source file count per language
}

// AnalysisResult holds the output of a single analysis pass (Phase 2).
type AnalysisResult struct {
	Name     string                 // Name of the analysis (e.g., "complexity")
	Category string                 // Category identifier (e.g., "C1", "C3", "C6")
	Metrics  map[string]interface{} // Analysis metrics
}

// MetricSummary holds avg/max for a numeric metric.
type MetricSummary struct {
	Avg       float64
	Max       int
	MaxEntity string // which function/file has the max
}

// FunctionMetric holds per-function analysis data.
type FunctionMetric struct {
	Package    string
	Name       string
	File       string
	Line       int
	Complexity int
	LineCount  int
}

// DuplicateBlock represents a detected code clone.
type DuplicateBlock struct {
	FileA     string
	StartA    int
	EndA      int
	FileB     string
	StartB    int
	EndB      int
	LineCount int
}

// C1Metrics holds Code Health metric results.
type C1Metrics struct {
	CyclomaticComplexity MetricSummary
	FunctionLength       MetricSummary
	FileSize             MetricSummary
	AfferentCoupling     map[string]int   // pkg path -> incoming dep count
	EfferentCoupling     map[string]int   // pkg path -> outgoing dep count
	DuplicationRate      float64          // percentage 0-100
	DuplicatedBlocks     []DuplicateBlock
	Functions            []FunctionMetric // per-function detail
}

// C3Metrics holds Architectural Navigability metric results.
type C3Metrics struct {
	MaxDirectoryDepth int
	AvgDirectoryDepth float64
	ModuleFanout      MetricSummary // avg refs per module
	CircularDeps      [][]string    // each cycle as list of package paths
	ImportComplexity  MetricSummary // avg relative path segments
	DeadExports       []DeadExport  // unreferenced exported symbols
}

// DeadExport represents an exported symbol not referenced within the module.
type DeadExport struct {
	Package string
	Name    string
	File    string
	Line    int
	Kind    string // "func", "type", "var", "const"
}

// C2Metrics holds Semantic Explicitness metric results.
type C2Metrics struct {
	PerLanguage map[Language]*C2LanguageMetrics
	Aggregate   *C2LanguageMetrics // LOC-weighted aggregate
}

// C2LanguageMetrics holds C2 metrics for a single language.
type C2LanguageMetrics struct {
	TypeAnnotationCoverage float64 // % of functions/params with type annotations (0-100)
	NamingConsistency      float64 // % of identifiers following convention (0-100)
	MagicNumberRatio       float64 // magic numbers per 1000 LOC
	TypeStrictness         float64 // 0 or 1: strict mode on/off (Python mypy, TS strict)
	NullSafety             float64 // % of pointer/nullable usages with safety checks (0-100)
	TotalFunctions         int     // total functions analyzed
	TotalIdentifiers       int     // total identifiers checked for naming
	MagicNumberCount       int     // raw count of magic numbers
	LOC                    int     // lines of code for this language
}

// C6Metrics holds Testing Infrastructure metric results.
type C6Metrics struct {
	TestFileCount    int
	SourceFileCount  int
	TestToCodeRatio  float64          // test LOC / source LOC
	CoveragePercent  float64          // -1 if not available
	CoverageSource   string           // "go-cover", "lcov", "cobertura", "none"
	TestIsolation    float64          // percentage of tests without external deps
	AssertionDensity MetricSummary    // assertions per test function
	TestFunctions    []TestFunctionMetric
}

// TestFunctionMetric holds per-test-function data.
type TestFunctionMetric struct {
	Package        string
	Name           string
	File           string
	Line           int
	AssertionCount int
	HasExternalDep bool
}

// C5Metrics holds Temporal & Operational Dynamics metric results.
type C5Metrics struct {
	Available            bool
	ChurnRate            float64       // avg lines changed per commit (90-day window)
	TemporalCouplingPct  float64       // % of file pairs with >70% co-change rate
	AuthorFragmentation  float64       // avg distinct authors per file (90-day window)
	CommitStability      float64       // median days between changes per file
	HotspotConcentration float64       // % of total changes in top 10% of files
	TopHotspots          []FileChurn   // top churning files (up to 10)
	CoupledPairs         []CoupledPair // detected temporal couplings
	TotalCommits         int
	TimeWindowDays       int
}

// FileChurn holds churn data for a single file.
type FileChurn struct {
	Path         string
	TotalChanges int
	CommitCount  int
	AuthorCount  int
}

// CoupledPair holds a pair of files with temporal coupling.
type CoupledPair struct {
	FileA         string
	FileB         string
	Coupling      float64 // 0-100 percentage
	SharedCommits int
}

// C4Metrics holds Documentation Quality metric results.
type C4Metrics struct {
	Available           bool
	ReadmePresent       bool
	ReadmeWordCount     int
	CommentDensity      float64 // % lines with comments (0-100)
	APIDocCoverage      float64 // % public APIs with docstrings (0-100)
	ChangelogPresent    bool
	ChangelogDaysOld    int // -1 if not present
	DiagramsPresent     bool
	ExamplesPresent     bool
	ContributingPresent bool
	// Counts for verbose output
	TotalSourceLines int
	CommentLines     int
	PublicAPIs       int
	DocumentedAPIs   int

	// LLM-based metrics (only populated if --enable-c4-llm is used)
	LLMEnabled        bool    // true if LLM analysis was performed
	ReadmeClarity     int     // 1-10 scale
	ExampleQuality    int     // 1-10 scale
	Completeness      int     // 1-10 scale
	CrossRefCoherence int     // 1-10 scale
	LLMCostUSD        float64 // Actual cost incurred
	LLMTokensUsed     int     // Total tokens used
	LLMFilesSampled   int     // Number of files sampled for LLM analysis
}

// C7Metrics holds Agent Evaluation metric results including 5 MECE metrics.
type C7Metrics struct {
	Available bool // false if claude CLI not found or user declined

	// Legacy 4-task scores (0-100 scale) - preserved for backward compatibility
	IntentClarity          int // 0-100 score
	ModificationConfidence int // 0-100 score
	CrossFileCoherence     int // 0-100 score
	SemanticCompleteness   int // 0-100 score

	// NEW: 5 MECE metrics (1-10 scale)
	TaskExecutionConsistency       int // M1: Reproducibility across runs (1-10)
	CodeBehaviorComprehension      int // M2: Understanding what code does (1-10)
	CrossFileNavigation            int // M3: Tracing dependencies across files (1-10)
	IdentifierInterpretability     int // M4: Inferring meaning from names (1-10)
	DocumentationAccuracyDetection int // M5: Detecting comment/code mismatches (1-10)

	// Aggregate scores
	OverallScore float64 // Legacy: average of 4 task scores (0-100)
	MECEScore    float64 // NEW: weighted average of 5 MECE metrics (1-10)

	// Detailed results
	TaskResults   []C7TaskResult   // Legacy task results
	MetricResults []C7MetricResult // NEW: MECE metric results

	// Execution metadata
	TotalDuration float64 // seconds
	TokensUsed    int     // estimated total tokens
	CostUSD       float64 // estimated cost
}

// C7TaskResult holds results for a single C7 evaluation task.
type C7TaskResult struct {
	TaskID    string  // e.g., "intent_clarity"
	TaskName  string  // e.g., "Intent Clarity"
	Score     int     // 0-100
	Status    string  // completed, timeout, error
	Duration  float64 // seconds
	Reasoning string  // scoring rationale from LLM judge
}

// C7MetricResult holds results for a single MECE metric.
type C7MetricResult struct {
	MetricID     string           // e.g., "task_execution_consistency"
	MetricName   string           // e.g., "Task Execution Consistency"
	Score        int              // 1-10
	Status       string           // completed, timeout, error
	Duration     float64          // seconds
	Reasoning    string           // scoring rationale
	Samples      []string         // sample descriptions used
	DebugSamples []C7DebugSample  `json:"debug_samples,omitempty"` // only present when debug active
}

// C7IndicatorMatch records one heuristic indicator check during scoring.
type C7IndicatorMatch struct {
	Name    string `json:"name"`    // e.g., "positive:returns", "negative:unclear"
	Matched bool   `json:"matched"` // Whether the indicator was found
	Delta   int    `json:"delta"`   // Point contribution (+1, -1, +2, etc.)
}

// C7ScoreTrace records the complete scoring breakdown for one sample.
type C7ScoreTrace struct {
	BaseScore  int                `json:"base_score"`  // Starting score before adjustments
	Indicators []C7IndicatorMatch `json:"indicators"`  // Each indicator checked
	FinalScore int                `json:"final_score"` // Score after clamping to 1-10
}

// C7DebugSample holds complete debug data for one metric sample evaluation.
// Only populated when debug mode is active.
type C7DebugSample struct {
	FilePath    string       `json:"file_path"`
	Description string       `json:"description"`
	Prompt      string       `json:"prompt"`
	Response    string       `json:"response"`
	Score       int          `json:"score"`
	Duration    float64      `json:"duration_seconds"`
	ScoreTrace  C7ScoreTrace `json:"score_trace"`
	Error       string       `json:"error,omitempty"`
}
