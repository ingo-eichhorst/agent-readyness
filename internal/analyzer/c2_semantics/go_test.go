package c2

import (
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestC2GoAnalyzer_SelfAnalysis(t *testing.T) {
	// Parse this project's own codebase for C2 analysis
	p := &parser.GoPackagesParser{}
	pkgs, err := p.Parse("../../../")
	if err != nil {
		t.Fatalf("failed to parse project: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages found")
	}

	analyzer := &C2GoAnalyzer{pkgs: pkgs}
	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  "../../../",
	}

	metrics, err := analyzer.Analyze(target)
	if err != nil {
		t.Fatalf("C2GoAnalyzer.Analyze() error: %v", err)
	}

	// Go is statically typed: TypeAnnotationCoverage must be 100
	if metrics.TypeAnnotationCoverage != 100 {
		t.Errorf("TypeAnnotationCoverage = %v, want 100 (Go is statically typed)", metrics.TypeAnnotationCoverage)
	}

	// Go has compile-time type checking: TypeStrictness must be 1
	if metrics.TypeStrictness != 1 {
		t.Errorf("TypeStrictness = %v, want 1 (Go has compile-time type checking)", metrics.TypeStrictness)
	}

	// NamingConsistency should be > 0 (our codebase follows Go conventions)
	if metrics.NamingConsistency <= 0 {
		t.Errorf("NamingConsistency = %v, want > 0", metrics.NamingConsistency)
	}
	// Our codebase should have reasonably high naming consistency
	if metrics.NamingConsistency < 70 {
		t.Errorf("NamingConsistency = %v, want > 70 for well-structured Go code", metrics.NamingConsistency)
	}

	// MagicNumberRatio should be >= 0
	if metrics.MagicNumberRatio < 0 {
		t.Errorf("MagicNumberRatio = %v, want >= 0", metrics.MagicNumberRatio)
	}

	// NullSafety should be between 0 and 100
	if metrics.NullSafety < 0 || metrics.NullSafety > 100 {
		t.Errorf("NullSafety = %v, want between 0 and 100", metrics.NullSafety)
	}

	// TotalFunctions should be positive
	if metrics.TotalFunctions <= 0 {
		t.Errorf("TotalFunctions = %d, want > 0", metrics.TotalFunctions)
	}

	// TotalIdentifiers should be positive
	if metrics.TotalIdentifiers <= 0 {
		t.Errorf("TotalIdentifiers = %d, want > 0", metrics.TotalIdentifiers)
	}

	// LOC should be positive
	if metrics.LOC <= 0 {
		t.Errorf("LOC = %d, want > 0", metrics.LOC)
	}

	t.Logf("C2 Go metrics: TypeAnnotation=%.0f NamingConsistency=%.1f MagicNumberRatio=%.2f TypeStrictness=%.0f NullSafety=%.1f Functions=%d Identifiers=%d MagicNumbers=%d LOC=%d",
		metrics.TypeAnnotationCoverage, metrics.NamingConsistency, metrics.MagicNumberRatio,
		metrics.TypeStrictness, metrics.NullSafety,
		metrics.TotalFunctions, metrics.TotalIdentifiers, metrics.MagicNumberCount, metrics.LOC)
}

func TestC2Analyzer_GoTarget(t *testing.T) {
	p := &parser.GoPackagesParser{}
	pkgs, err := p.Parse("../../../")
	if err != nil {
		t.Fatalf("failed to parse project: %v", err)
	}

	analyzer := &C2Analyzer{}
	analyzer.SetGoPackages(pkgs)

	targets := []*types.AnalysisTarget{
		{
			Language: types.LangGo,
			RootDir:  "../../../",
		},
	}

	result, err := analyzer.Analyze(targets)
	if err != nil {
		t.Fatalf("C2Analyzer.Analyze() error: %v", err)
	}

	if result.Category != "C2" {
		t.Errorf("Category = %q, want C2", result.Category)
	}

	raw, ok := result.Metrics["c2"]
	if !ok {
		t.Fatal("missing c2 key in Metrics")
	}

	c2, ok := raw.(*types.C2Metrics)
	if !ok {
		t.Fatal("c2 metric is not *types.C2Metrics")
	}

	if c2.Aggregate == nil {
		t.Fatal("Aggregate is nil")
	}

	if c2.Aggregate.TypeAnnotationCoverage != 100 {
		t.Errorf("Aggregate.TypeAnnotationCoverage = %v, want 100", c2.Aggregate.TypeAnnotationCoverage)
	}

	// Should have Go in PerLanguage
	goMetrics, ok := c2.PerLanguage[types.LangGo]
	if !ok {
		t.Fatal("missing Go in PerLanguage")
	}
	if goMetrics.LOC <= 0 {
		t.Errorf("PerLanguage[Go].LOC = %d, want > 0", goMetrics.LOC)
	}
}

func TestC2Analyzer_EmptyTargets(t *testing.T) {
	analyzer := &C2Analyzer{}

	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("C2Analyzer.Analyze(nil) error: %v", err)
	}

	if result.Category != "C2" {
		t.Errorf("Category = %q, want C2", result.Category)
	}

	c2 := result.Metrics["c2"].(*types.C2Metrics)
	if c2.Aggregate != nil {
		t.Errorf("Aggregate should be nil for empty targets, got %+v", c2.Aggregate)
	}
}

func TestC2Analyzer_Name(t *testing.T) {
	analyzer := &C2Analyzer{}
	if got := analyzer.Name(); got != "C2: Semantic Explicitness" {
		t.Errorf("Name() = %q, want %q", got, "C2: Semantic Explicitness")
	}
}

func TestIsConsistentGoName(t *testing.T) {
	tests := []struct {
		name     string
		exported bool
		want     bool
	}{
		{"ParsedPackage", true, true},
		{"parsedPackage", false, true},
		{"GoPackagesParser", true, true},
		{"goPackagesParser", false, true},
		{"snake_case", false, false},
		{"Snake_Case", true, false},
		{"TestHello_World", true, true}, // Test functions can have underscores
		{"x", false, true},              // single letter -- but won't be called, shouldSkipName skips it
		{"ID", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConsistentGoName(tt.name, tt.exported)
			if got != tt.want {
				t.Errorf("isConsistentGoName(%q, %v) = %v, want %v", tt.name, tt.exported, got, tt.want)
			}
		})
	}
}

func TestIsCommonNumericLiteral(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"0", true},
		{"1", true},
		{"2", true},
		{"0.0", true},
		{"1.0", true},
		{"3", false},
		{"42", false},
		{"100", false},
		{"0xFF", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := isCommonNumericLiteral(tt.value)
			if got != tt.want {
				t.Errorf("isCommonNumericLiteral(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestShouldSkipName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"_", true},
		{"x", true},
		{"ID", true},
		{"URL", true},
		{"hello", false},
		{"MyFunc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipName(tt.name)
			if got != tt.want {
				t.Errorf("shouldSkipName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
