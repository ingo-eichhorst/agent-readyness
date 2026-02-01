package pipeline

import (
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Parser loads and parses Go packages from a module directory.
// Kept for Go parser compatibility; will be deprecated when multi-parser replaces it.
type Parser interface {
	Parse(rootDir string) ([]*parser.ParsedPackage, error)
}

// Analyzer runs a specific analysis pass over analysis targets.
// Targets are language-agnostic; analyzers that need Go-specific data
// should also implement GoAwareAnalyzer.
type Analyzer interface {
	Name() string
	Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error)
}

// GoAwareAnalyzer is an Analyzer that also needs access to Go-specific
// parsed packages (ASTs, type info). The pipeline calls SetGoPackages
// before Analyze for any analyzer implementing this interface.
type GoAwareAnalyzer interface {
	Analyzer
	SetGoPackages(pkgs []*parser.ParsedPackage)
}

// StubAnalyzer is a no-op analyzer that returns an empty result.
// Used as a placeholder when no real analyzers are configured.
type StubAnalyzer struct{}

// Name returns the analyzer name.
func (s *StubAnalyzer) Name() string {
	return "stub"
}

// Analyze returns an empty AnalysisResult.
func (s *StubAnalyzer) Analyze(_ []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	return &types.AnalysisResult{
		Name:    "stub",
		Metrics: make(map[string]interface{}),
	}, nil
}
