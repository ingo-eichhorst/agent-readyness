package pipeline

import (
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Parser loads and parses Go packages from a module directory.
// Phase 1 used StubParser; Phase 2 uses GoPackagesParser.
type Parser interface {
	Parse(rootDir string) ([]*parser.ParsedPackage, error)
}

// Analyzer runs a specific analysis pass over parsed packages.
// Phase 1 used StubAnalyzer; Phase 2 plugs in real implementations.
type Analyzer interface {
	Name() string
	Analyze(pkgs []*parser.ParsedPackage) (*types.AnalysisResult, error)
}

// StubParser is a no-op parser that returns an empty package slice.
// Used as fallback when no real parser is configured.
type StubParser struct{}

// Parse returns an empty ParsedPackage slice without loading anything.
func (s *StubParser) Parse(rootDir string) ([]*parser.ParsedPackage, error) {
	return []*parser.ParsedPackage{}, nil
}

// StubAnalyzer is a no-op analyzer that returns an empty result.
// Used as a placeholder when no real analyzers are configured.
type StubAnalyzer struct{}

// Name returns the analyzer name.
func (s *StubAnalyzer) Name() string {
	return "stub"
}

// Analyze returns an empty AnalysisResult.
func (s *StubAnalyzer) Analyze(pkgs []*parser.ParsedPackage) (*types.AnalysisResult, error) {
	return &types.AnalysisResult{
		Name:    "stub",
		Metrics: make(map[string]interface{}),
	}, nil
}
