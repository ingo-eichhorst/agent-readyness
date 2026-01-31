package analyzer

import (
	"testing"

	"github.com/ingo/agent-readyness/pkg/types"
)

func c3Metrics(t *testing.T, result *types.AnalysisResult) *types.C3Metrics {
	t.Helper()
	raw, ok := result.Metrics["c3"]
	if !ok {
		t.Fatal("C3 metrics not found in result.Metrics[\"c3\"]")
	}
	m, ok := raw.(*types.C3Metrics)
	if !ok {
		t.Fatalf("Metrics[\"c3\"] is %T, want *types.C3Metrics", raw)
	}
	return m
}

// TestC3DirectoryDepth verifies max and avg directory depth relative to module root.
func TestC3DirectoryDepth(t *testing.T) {
	pkgs := loadTestPackages(t, "deepnest")

	analyzer := &C3Analyzer{}
	result, err := analyzer.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := c3Metrics(t, result)

	// deepnest has root (depth 0) and a/b/c/d (depth 4)
	if m.MaxDirectoryDepth != 4 {
		t.Errorf("MaxDirectoryDepth = %d, want 4", m.MaxDirectoryDepth)
	}
	// avg of 0 and 4 = 2.0
	if m.AvgDirectoryDepth != 2.0 {
		t.Errorf("AvgDirectoryDepth = %f, want 2.0", m.AvgDirectoryDepth)
	}
}

// TestC3ModuleFanout verifies average and max intra-module import counts.
func TestC3ModuleFanout(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := &C3Analyzer{}
	result, err := analyzer.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := c3Metrics(t, result)

	// coupling: pkga imports pkgb (fanout 1), pkgb imports nothing (fanout 0)
	// Max fanout = 1
	if m.ModuleFanout.Max != 1 {
		t.Errorf("ModuleFanout.Max = %d, want 1", m.ModuleFanout.Max)
	}
	// Avg fanout: (1+0)/2 = 0.5
	if m.ModuleFanout.Avg < 0.4 || m.ModuleFanout.Avg > 0.6 {
		t.Errorf("ModuleFanout.Avg = %f, want ~0.5", m.ModuleFanout.Avg)
	}
}

// TestC3CircularDeps verifies that no cycles are found in valid Go code.
func TestC3CircularDeps(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := &C3Analyzer{}
	result, err := analyzer.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := c3Metrics(t, result)

	// Valid Go code cannot have circular imports (compiler prevents it).
	if len(m.CircularDeps) != 0 {
		t.Errorf("CircularDeps = %v, want empty (valid Go code)", m.CircularDeps)
	}
}

// TestC3ImportComplexity verifies average path segment count for intra-module imports.
func TestC3ImportComplexity(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := &C3Analyzer{}
	result, err := analyzer.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := c3Metrics(t, result)

	// coupling: pkga imports "github.com/ingo/agent-readyness/testdata/coupling/pkgb"
	// Relative to module root "github.com/ingo/agent-readyness/testdata/coupling" -> "pkgb" = 1 segment
	// Only one intra-module import, so avg = 1.0, max = 1
	if m.ImportComplexity.Avg < 0.9 || m.ImportComplexity.Avg > 1.1 {
		t.Errorf("ImportComplexity.Avg = %f, want ~1.0", m.ImportComplexity.Avg)
	}
	if m.ImportComplexity.Max != 1 {
		t.Errorf("ImportComplexity.Max = %d, want 1", m.ImportComplexity.Max)
	}
}

// TestC3DeadCode verifies that unreferenced exported symbols are detected.
func TestC3DeadCode(t *testing.T) {
	pkgs := loadTestPackages(t, "deadcode")

	analyzer := &C3Analyzer{}
	result, err := analyzer.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := c3Metrics(t, result)

	// lib.ExportedUnused and lib.UnusedType should be dead
	// lib.ExportedUsed is referenced by user package -> alive
	deadNames := make(map[string]bool)
	for _, d := range m.DeadExports {
		deadNames[d.Name] = true
	}

	if !deadNames["ExportedUnused"] {
		t.Error("ExportedUnused should be flagged as dead code")
	}
	if !deadNames["UnusedType"] {
		t.Error("UnusedType should be flagged as dead code")
	}
	if deadNames["ExportedUsed"] {
		t.Error("ExportedUsed should NOT be flagged as dead code")
	}
}

// TestC3AnalyzerInterface verifies that C3Analyzer implements the Name/Analyze contract.
func TestC3AnalyzerInterface(t *testing.T) {
	a := &C3Analyzer{}
	if a.Name() != "C3: Architecture" {
		t.Errorf("Name() = %q, want %q", a.Name(), "C3: Architecture")
	}

	pkgs := loadTestPackages(t, "deepnest")
	result, err := a.Analyze(pkgs)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if result.Name != "C3: Architecture" {
		t.Errorf("result.Name = %q, want %q", result.Name, "C3: Architecture")
	}
	if result.Category != "C3" {
		t.Errorf("result.Category = %q, want %q", result.Category, "C3")
	}
}
