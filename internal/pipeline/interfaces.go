package pipeline

import "github.com/ingo/agent-readyness/pkg/types"

// Parser converts discovered files into parsed files with AST information.
// Phase 1 uses StubParser; Phase 2 plugs in real implementations.
type Parser interface {
	Parse(files []types.DiscoveredFile) ([]types.ParsedFile, error)
}

// Analyzer runs a specific analysis pass over parsed files.
// Phase 1 uses StubAnalyzer; Phase 2 plugs in real implementations.
type Analyzer interface {
	Name() string
	Analyze(files []types.ParsedFile) (*types.AnalysisResult, error)
}

// StubParser is a passthrough parser that converts DiscoveredFiles to ParsedFiles
// without any AST processing. Used in Phase 1.
type StubParser struct{}

// Parse converts each DiscoveredFile to a ParsedFile by copying path and class fields.
func (s *StubParser) Parse(files []types.DiscoveredFile) ([]types.ParsedFile, error) {
	parsed := make([]types.ParsedFile, len(files))
	for i, f := range files {
		parsed[i] = types.ParsedFile{
			Path:    f.Path,
			RelPath: f.RelPath,
			Class:   f.Class,
		}
	}
	return parsed, nil
}

// StubAnalyzer is a no-op analyzer that returns an empty result.
// Used in Phase 1 as a placeholder.
type StubAnalyzer struct{}

// Name returns the analyzer name.
func (s *StubAnalyzer) Name() string {
	return "stub"
}

// Analyze returns an empty AnalysisResult.
func (s *StubAnalyzer) Analyze(files []types.ParsedFile) (*types.AnalysisResult, error) {
	return &types.AnalysisResult{
		Name:    "stub",
		Metrics: make(map[string]interface{}),
	}, nil
}
