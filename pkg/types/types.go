package types

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
	Files          []DiscoveredFile // All discovered files
}

// AnalysisResult holds the output of a single analysis pass (Phase 2).
type AnalysisResult struct {
	Name    string                 // Name of the analysis (e.g., "complexity")
	Metrics map[string]interface{} // Analysis metrics
}
