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
}

// ParsedFile represents a file after AST parsing (Phase 2).
type ParsedFile struct {
	Path    string    // Absolute path to the file
	RelPath string    // Path relative to project root
	Class   FileClass // Classification of the file
}

// ScanResult holds the output of the file discovery phase.
type ScanResult struct {
	RootDir        string           // Absolute path to project root
	TotalFiles     int              // Total files discovered
	SourceCount    int              // Number of source files
	TestCount      int              // Number of test files
	GeneratedCount int              // Number of generated files
	VendorCount    int              // Number of vendor-excluded files
	GitignoreCount int              // Number of gitignore-excluded files
	SkippedCount   int              // Files/dirs skipped due to errors (permission denied, broken paths, etc.)
	SymlinkCount   int              // Symlinks detected and skipped
	Files          []DiscoveredFile // All discovered files
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
