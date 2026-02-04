// Package analyzer provides code analysis implementations for the ARS pipeline.
// This file provides type aliases and constructor wrappers for backward compatibility
// after analyzers are reorganized into category subdirectories.
package analyzer

import (
	c1 "github.com/ingo/agent-readyness/internal/analyzer/c1_code_quality"
	c2 "github.com/ingo/agent-readyness/internal/analyzer/c2_semantics"
	c3 "github.com/ingo/agent-readyness/internal/analyzer/c3_architecture"
	c4 "github.com/ingo/agent-readyness/internal/analyzer/c4_documentation"
	c5 "github.com/ingo/agent-readyness/internal/analyzer/c5_temporal"
	c6 "github.com/ingo/agent-readyness/internal/analyzer/c6_testing"
	c7 "github.com/ingo/agent-readyness/internal/analyzer/c7_agent"
	"github.com/ingo/agent-readyness/internal/parser"
)

// Type aliases for backward compatibility.
// These allow existing code to continue using analyzer.CxAnalyzer
// after the types are moved to subdirectories.
type C1Analyzer = c1.C1Analyzer
type C2Analyzer = c2.C2Analyzer
type C3Analyzer = c3.C3Analyzer
type C4Analyzer = c4.C4Analyzer
type C5Analyzer = c5.C5Analyzer
type C6Analyzer = c6.C6Analyzer
type C7Analyzer = c7.C7Analyzer

// Constructor wrappers for backward compatibility.
// These allow existing code to continue calling analyzer.NewCxAnalyzer()
// after the constructors are moved to subdirectories.

// NewC1Analyzer creates a C1 (Code Health) analyzer.
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
	return c1.NewC1Analyzer(tsParser)
}

// NewC2Analyzer creates a C2 (Semantic Explicitness) analyzer.
func NewC2Analyzer(tsParser *parser.TreeSitterParser) *C2Analyzer {
	return c2.NewC2Analyzer(tsParser)
}

// NewC3Analyzer creates a C3 (Architecture) analyzer.
func NewC3Analyzer(tsParser *parser.TreeSitterParser) *C3Analyzer {
	return c3.NewC3Analyzer(tsParser)
}

// NewC4Analyzer creates a C4 (Documentation) analyzer.
func NewC4Analyzer(tsParser *parser.TreeSitterParser) *C4Analyzer {
	return c4.NewC4Analyzer(tsParser)
}

// NewC5Analyzer creates a C5 (Temporal/Git) analyzer.
func NewC5Analyzer() *C5Analyzer {
	return c5.NewC5Analyzer()
}

// NewC6Analyzer creates a C6 (Testing) analyzer.
func NewC6Analyzer(tsParser *parser.TreeSitterParser) *C6Analyzer {
	return c6.NewC6Analyzer(tsParser)
}

// NewC7Analyzer creates a C7 (Agent) analyzer.
func NewC7Analyzer() *C7Analyzer {
	return c7.NewC7Analyzer()
}
