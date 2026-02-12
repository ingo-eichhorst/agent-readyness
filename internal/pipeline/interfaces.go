package pipeline

import (
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// parseProvider loads and parses Go packages from a module directory.
// Kept for Go parser compatibility; will be deprecated when multi-parser replaces it.
type parseProvider interface {
	Parse(rootDir string) ([]*parser.ParsedPackage, error)
}

// analyzerIface runs a specific analysis pass over analysis targets.
// Targets are language-agnostic; analyzers that need Go-specific data
// should also implement goAwareAnalyzer.
type analyzerIface interface {
	Name() string
	Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error)
}

// goAwareAnalyzer is an analyzerIface that also needs access to Go-specific
// parsed packages (ASTs, type info). The pipeline calls SetGoPackages
// before Analyze for any analyzer implementing this interface.
type goAwareAnalyzer interface {
	analyzerIface
	SetGoPackages(pkgs []*parser.ParsedPackage)
}
