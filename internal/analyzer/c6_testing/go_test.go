package c6

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	projectparser "github.com/ingo-eichhorst/agent-readyness/internal/parser"
	pkgtypes "github.com/ingo-eichhorst/agent-readyness/pkg/types"
	"golang.org/x/tools/go/packages"
)

// buildTestPackages creates ParsedPackage slices from inline Go source code
// for deterministic testing without loading real packages.
func buildTestPackages(t *testing.T, srcFiles map[string]string, testFiles map[string]string) []*projectparser.ParsedPackage {
	t.Helper()

	fset := token.NewFileSet()
	var result []*projectparser.ParsedPackage

	// Build source package
	if len(srcFiles) > 0 {
		srcPkg := &projectparser.ParsedPackage{
			ID:      "example.com/pkg",
			Name:    "pkg",
			PkgPath: "example.com/pkg",
			Fset:    fset,
			Imports: make(map[string]*packages.Package),
		}
		for name, src := range srcFiles {
			f, err := goparser.ParseFile(fset, name, src, goparser.AllErrors)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			srcPkg.Syntax = append(srcPkg.Syntax, f)
			srcPkg.GoFiles = append(srcPkg.GoFiles, name)
		}
		result = append(result, srcPkg)
	}

	// Build test package
	if len(testFiles) > 0 {
		testPkg := &projectparser.ParsedPackage{
			ID:      "example.com/pkg [example.com/pkg.test]",
			Name:    "pkg",
			PkgPath: "example.com/pkg",
			Fset:    fset,
			ForTest: "example.com/pkg",
			Imports: make(map[string]*packages.Package),
		}
		for name, src := range testFiles {
			f, err := goparser.ParseFile(fset, name, src, goparser.AllErrors)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			testPkg.Syntax = append(testPkg.Syntax, f)
			testPkg.GoFiles = append(testPkg.GoFiles, name)
		}
		result = append(result, testPkg)
	}

	return result
}

// --- C6-01: Test Detection ---

func TestC6_TestDetection(t *testing.T) {
	pkgs := buildTestPackages(t,
		map[string]string{
			"foo.go": "package pkg\nfunc Foo() {}\n",
			"bar.go": "package pkg\nfunc Bar() {}\n",
		},
		map[string]string{
			"foo_test.go": "package pkg\nimport \"testing\"\nfunc TestFoo(t *testing.T) {}\n",
		},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m, ok := result.Metrics["c6"].(*pkgtypes.C6Metrics)
	if !ok {
		t.Fatal("expected C6Metrics in result")
	}

	if m.TestFileCount != 1 {
		t.Errorf("TestFileCount = %d, want 1", m.TestFileCount)
	}
	if m.SourceFileCount != 2 {
		t.Errorf("SourceFileCount = %d, want 2", m.SourceFileCount)
	}
}

// --- C6-02: Test-to-Code Ratio ---

func TestC6_TestToCodeRatio(t *testing.T) {
	// Source: 10 lines, Test: 5 lines -> ratio 0.5
	srcCode := "package pkg\n\nfunc Foo() {\n\tx := 1\n\ty := 2\n\tz := x + y\n\t_ = z\n}\n\nfunc Bar() {}\n"
	testCode := "package pkg\n\nimport \"testing\"\n\nfunc TestFoo(t *testing.T) {}\n"

	pkgs := buildTestPackages(t,
		map[string]string{"foo.go": srcCode},
		map[string]string{"foo_test.go": testCode},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := result.Metrics["c6"].(*pkgtypes.C6Metrics)

	// Source is 10 lines, test is 5 lines -> ratio = 5/10 = 0.5
	if m.TestToCodeRatio < 0.4 || m.TestToCodeRatio > 0.6 {
		t.Errorf("TestToCodeRatio = %f, want ~0.5", m.TestToCodeRatio)
	}
}

// --- C6-03: Coverage Parsing ---

func TestC6_GoCoverageProfile(t *testing.T) {
	a := &C6Analyzer{}

	coverFile := filepath.Join("..", "..", "..", "testdata", "coverage", "cover.out")
	if _, err := os.Stat(coverFile); os.IsNotExist(err) {
		t.Skip("cover.out testdata not found")
	}

	pct, src, err := a.parseCoverage(filepath.Join("..", "..", "..", "testdata", "coverage"))
	if err != nil {
		t.Fatalf("parseCoverage: %v", err)
	}

	if src != "go-cover" {
		t.Errorf("source = %q, want go-cover", src)
	}

	// cover.out: 4 statements, 2 covered -> 50%
	if pct < 49.0 || pct > 51.0 {
		t.Errorf("coverage = %f, want ~50.0", pct)
	}
}

func TestC6_LCOVParsing(t *testing.T) {
	a := &C6Analyzer{}

	// Use a temp dir with only lcov.info (no cover.out)
	tmpDir := t.TempDir()
	lcovData, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "coverage", "lcov.info"))
	if err != nil {
		t.Fatalf("read lcov fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "lcov.info"), lcovData, 0644); err != nil {
		t.Fatalf("write lcov: %v", err)
	}

	pct, src, err := a.parseCoverage(tmpDir)
	if err != nil {
		t.Fatalf("parseCoverage: %v", err)
	}

	if src != "lcov" {
		t.Errorf("source = %q, want lcov", src)
	}

	// lcov: 6 total lines, 4 hit -> 66.67%
	if pct < 66.0 || pct > 67.0 {
		t.Errorf("coverage = %f, want ~66.67", pct)
	}
}

func TestC6_CoberturaParsing(t *testing.T) {
	a := &C6Analyzer{}

	// Use a temp dir with only cobertura.xml
	tmpDir := t.TempDir()
	cobData, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "coverage", "cobertura.xml"))
	if err != nil {
		t.Fatalf("read cobertura fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cobertura.xml"), cobData, 0644); err != nil {
		t.Fatalf("write cobertura: %v", err)
	}

	pct, src, err := a.parseCoverage(tmpDir)
	if err != nil {
		t.Fatalf("parseCoverage: %v", err)
	}

	if src != "cobertura" {
		t.Errorf("source = %q, want cobertura", src)
	}

	// cobertura: line-rate="0.75" -> 75%
	if pct < 74.0 || pct > 76.0 {
		t.Errorf("coverage = %f, want ~75.0", pct)
	}
}

func TestC6_NoCoverageFile(t *testing.T) {
	a := &C6Analyzer{}

	tmpDir := t.TempDir()
	pct, src, err := a.parseCoverage(tmpDir)
	if err != nil {
		t.Fatalf("parseCoverage: %v", err)
	}

	if src != "none" {
		t.Errorf("source = %q, want none", src)
	}
	if pct != -1 {
		t.Errorf("coverage = %f, want -1", pct)
	}
}

// --- C6-04: Test Isolation ---

func TestC6_TestIsolation(t *testing.T) {
	// Two test functions: one isolated, one with net/http import
	pkgs := buildTestPackages(t,
		map[string]string{
			"foo.go": "package pkg\nfunc Foo() {}\n",
		},
		map[string]string{
			"foo_test.go": `package pkg

import "testing"

func TestIsolated(t *testing.T) {}
`,
			"bar_test.go": `package pkg

import (
	"testing"
	"net/http"
)

func TestWithHTTP(t *testing.T) {
	_ = http.DefaultClient
}
`,
		},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := result.Metrics["c6"].(*pkgtypes.C6Metrics)

	// 1 isolated out of 2 -> 50%
	if m.TestIsolation < 49.0 || m.TestIsolation > 51.0 {
		t.Errorf("TestIsolation = %f, want ~50.0", m.TestIsolation)
	}
}

// --- C6-05: Assertion Density ---

func TestC6_AssertionDensity(t *testing.T) {
	pkgs := buildTestPackages(t,
		map[string]string{
			"foo.go": "package pkg\nfunc Foo() int { return 1 }\n",
		},
		map[string]string{
			"foo_test.go": `package pkg

import "testing"

func TestFoo(t *testing.T) {
	t.Error("one")
	t.Errorf("two")
	t.Fatal("three")
}

func TestBar(t *testing.T) {
	t.Error("single")
}
`,
		},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := result.Metrics["c6"].(*pkgtypes.C6Metrics)

	// TestFoo has 3 assertions, TestBar has 1 -> avg 2.0, max 3
	if m.AssertionDensity.Avg < 1.9 || m.AssertionDensity.Avg > 2.1 {
		t.Errorf("AssertionDensity.Avg = %f, want ~2.0", m.AssertionDensity.Avg)
	}
	if m.AssertionDensity.Max != 3 {
		t.Errorf("AssertionDensity.Max = %d, want 3", m.AssertionDensity.Max)
	}
	if m.AssertionDensity.MaxEntity != "TestFoo" {
		t.Errorf("AssertionDensity.MaxEntity = %q, want TestFoo", m.AssertionDensity.MaxEntity)
	}

	// Verify test function details
	if len(m.TestFunctions) != 2 {
		t.Fatalf("TestFunctions count = %d, want 2", len(m.TestFunctions))
	}
}

func TestC6_TestifyAssertions(t *testing.T) {
	// Test that testify-style assert.Equal calls are counted
	pkgs := buildTestPackages(t,
		map[string]string{
			"foo.go": "package pkg\nfunc Foo() int { return 1 }\n",
		},
		map[string]string{
			"foo_test.go": `package pkg

import "testing"

// Simulating testify-style: variable named assert with methods
func TestWithAssert(t *testing.T) {
	assert := struct{ Equal func(a, b int); NotNil func(a int) }{
		Equal: func(a, b int) {},
		NotNil: func(a int) {},
	}
	assert.Equal(1, 1)
	assert.NotNil(1)
}
`,
		},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	m := result.Metrics["c6"].(*pkgtypes.C6Metrics)

	// assert.Equal + assert.NotNil = 2
	if m.AssertionDensity.Max != 2 {
		t.Errorf("AssertionDensity.Max = %d, want 2", m.AssertionDensity.Max)
	}
}

// --- C6 Analyzer interface ---

func TestC6_Name(t *testing.T) {
	a := &C6Analyzer{}
	if a.Name() != "C6: Testing" {
		t.Errorf("Name() = %q, want %q", a.Name(), "C6: Testing")
	}
}

func TestC6_ResultCategory(t *testing.T) {
	pkgs := buildTestPackages(t,
		map[string]string{"foo.go": "package pkg\nfunc Foo() {}\n"},
		map[string]string{"foo_test.go": "package pkg\nimport \"testing\"\nfunc TestFoo(t *testing.T) {}\n"},
	)

	a := &C6Analyzer{}
	a.SetGoPackages(pkgs)
	result, err := a.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if result.Category != "C6" {
		t.Errorf("Category = %q, want C6", result.Category)
	}
}

// Ensure unused imports satisfy compiler
var _ *ast.File
var _ *types.Package
