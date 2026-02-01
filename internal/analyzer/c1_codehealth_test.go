package analyzer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
)

// testdataDir returns the absolute path to the project testdata directory.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

// loadTestPackages loads Go packages from a testdata subdirectory.
func loadTestPackages(t *testing.T, subdir string) []*parser.ParsedPackage {
	t.Helper()
	p := &parser.GoPackagesParser{}
	pkgs, err := p.Parse(filepath.Join(testdataDir(), subdir))
	if err != nil {
		t.Fatalf("failed to parse %s: %v", subdir, err)
	}
	return pkgs
}

// --- C1-01: Cyclomatic Complexity ---

func TestC1_CyclomaticComplexity(t *testing.T) {
	pkgs := loadTestPackages(t, "complexity")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics, ok := result.Metrics["c1"].(*C1MetricsResult)
	if !ok {
		t.Fatal("expected C1MetricsResult in Metrics[\"c1\"]")
	}

	// Find functions by name in the metrics
	funcComplexity := make(map[string]int)
	for _, fm := range metrics.Functions {
		funcComplexity[fm.Name] = fm.Complexity
	}

	// SimpleFunc: no branches -> complexity 1
	if c, ok := funcComplexity["SimpleFunc"]; !ok || c != 1 {
		t.Errorf("SimpleFunc complexity = %d, want 1", c)
	}

	// OneBranch: 1 if -> complexity 2
	if c, ok := funcComplexity["OneBranch"]; !ok || c != 2 {
		t.Errorf("OneBranch complexity = %d, want 2", c)
	}

	// MultiBranch: if + for + 3 cases -> complexity 6
	if c, ok := funcComplexity["MultiBranch"]; !ok || c != 6 {
		t.Errorf("MultiBranch complexity = %d, want 6", c)
	}

	// Avg complexity should be computed
	if metrics.CyclomaticComplexity.Avg <= 0 {
		t.Errorf("CyclomaticComplexity.Avg = %f, want > 0", metrics.CyclomaticComplexity.Avg)
	}

	// Max should be MultiBranch
	if metrics.CyclomaticComplexity.Max < 6 {
		t.Errorf("CyclomaticComplexity.Max = %d, want >= 6", metrics.CyclomaticComplexity.Max)
	}
}

// --- C1-02: Function Length ---

func TestC1_FunctionLength(t *testing.T) {
	pkgs := loadTestPackages(t, "complexity")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics := result.Metrics["c1"].(*C1MetricsResult)

	funcLength := make(map[string]int)
	for _, fm := range metrics.Functions {
		funcLength[fm.Name] = fm.LineCount
	}

	// SimpleFunc: func decl + return + closing = 3 lines
	if l, ok := funcLength["SimpleFunc"]; !ok || l < 3 {
		t.Errorf("SimpleFunc LineCount = %d, want >= 3", l)
	}

	// FunctionLength summary should be populated
	if metrics.FunctionLength.Max <= 0 {
		t.Error("FunctionLength.Max should be > 0")
	}
	if metrics.FunctionLength.Avg <= 0 {
		t.Error("FunctionLength.Avg should be > 0")
	}
}

// --- C1-03: File Size ---

func TestC1_FileSize(t *testing.T) {
	pkgs := loadTestPackages(t, "complexity")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics := result.Metrics["c1"].(*C1MetricsResult)

	// The complexity/main.go fixture should have a known line count
	if metrics.FileSize.Max <= 0 {
		t.Error("FileSize.Max should be > 0")
	}
	if metrics.FileSize.Avg <= 0 {
		t.Error("FileSize.Avg should be > 0")
	}
}

// --- C1-04: Afferent Coupling ---

func TestC1_AfferentCoupling(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics := result.Metrics["c1"].(*C1MetricsResult)

	// pkgb is imported by pkga -> afferent(pkgb) = 1
	pkgbPath := "github.com/ingo/agent-readyness/testdata/coupling/pkgb"
	if ca, ok := metrics.AfferentCoupling[pkgbPath]; !ok || ca != 1 {
		t.Errorf("AfferentCoupling[pkgb] = %d, want 1", ca)
	}
}

// --- C1-05: Efferent Coupling ---

func TestC1_EfferentCoupling(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics := result.Metrics["c1"].(*C1MetricsResult)

	// pkga imports pkgb -> efferent(pkga) = 1
	pkgaPath := "github.com/ingo/agent-readyness/testdata/coupling/pkga"
	if ce, ok := metrics.EfferentCoupling[pkgaPath]; !ok || ce != 1 {
		t.Errorf("EfferentCoupling[pkga] = %d, want 1", ce)
	}

	// pkgb imports nothing intra-module -> efferent(pkgb) = 0
	pkgbPath := "github.com/ingo/agent-readyness/testdata/coupling/pkgb"
	if ce := metrics.EfferentCoupling[pkgbPath]; ce != 0 {
		t.Errorf("EfferentCoupling[pkgb] = %d, want 0", ce)
	}
}

// --- C1-06: Duplication ---

func TestC1_Duplication(t *testing.T) {
	pkgs := loadTestPackages(t, "duplication")

	analyzer := &C1Analyzer{}
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics := result.Metrics["c1"].(*C1MetricsResult)

	// Two identical 8-line blocks should be detected
	if len(metrics.DuplicatedBlocks) == 0 {
		t.Error("expected at least one duplicate block, got none")
	}

	// Duplication rate should be > 0
	if metrics.DuplicationRate <= 0 {
		t.Errorf("DuplicationRate = %f, want > 0", metrics.DuplicationRate)
	}
}

// --- Integration: Name and Category ---

func TestC1_NameAndCategory(t *testing.T) {
	analyzer := &C1Analyzer{}
	if analyzer.Name() != "C1: Code Health" {
		t.Errorf("Name() = %q, want %q", analyzer.Name(), "C1: Code Health")
	}
}
