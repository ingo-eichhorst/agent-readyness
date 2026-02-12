package c2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestC2PythonAnalyzer_ValidProject(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := newC2PythonAnalyzer(tsParser)

	// Build AnalysisTarget from testdata/valid-python-project
	testDir, err := filepath.Abs("../../../testdata/valid-python-project")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "app.py"),
				RelPath:  "app.py",
				Language: types.LangPython,
				Class:    types.ClassSource,
			},
		},
	}

	metrics, err := analyzer.Analyze(target)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// Type annotation coverage should be > 0 (app.py has type annotations)
	if metrics.TypeAnnotationCoverage <= 0 {
		t.Errorf("TypeAnnotationCoverage = %v, want > 0", metrics.TypeAnnotationCoverage)
	}

	// Naming consistency should be > 0 (app.py follows PEP 8)
	if metrics.NamingConsistency <= 0 {
		t.Errorf("NamingConsistency = %v, want > 0", metrics.NamingConsistency)
	}

	// TotalFunctions should be > 0
	if metrics.TotalFunctions <= 0 {
		t.Errorf("TotalFunctions = %d, want > 0", metrics.TotalFunctions)
	}

	// LOC should be > 0
	if metrics.LOC <= 0 {
		t.Errorf("LOC = %d, want > 0", metrics.LOC)
	}

	// Magic number ratio should be >= 0
	if metrics.MagicNumberRatio < 0 {
		t.Errorf("MagicNumberRatio = %v, want >= 0", metrics.MagicNumberRatio)
	}

	t.Logf("Python C2 metrics: TypeAnnotation=%.1f NamingConsistency=%.1f MagicNumberRatio=%.2f TypeStrictness=%.0f Functions=%d Identifiers=%d MagicNumbers=%d LOC=%d",
		metrics.TypeAnnotationCoverage, metrics.NamingConsistency, metrics.MagicNumberRatio,
		metrics.TypeStrictness,
		metrics.TotalFunctions, metrics.TotalIdentifiers, metrics.MagicNumberCount, metrics.LOC)
}

func TestC2PythonAnalyzer_TypeCheckerDetection(t *testing.T) {
	// Test with pyproject.toml that does NOT have [tool.mypy] section
	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")
	strictness := pyDetectTypeChecker(testDir)
	// Our test fixture doesn't have mypy config, so strictness should be 0
	if strictness != 0 {
		t.Errorf("pyDetectTypeChecker = %v, want 0 (no mypy/pyright config)", strictness)
	}

	// Create a temporary dir with mypy.ini to test detection
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "mypy.ini"), []byte("[mypy]\n"), 0644); err != nil {
		t.Fatalf("failed to write mypy.ini: %v", err)
	}
	strictness = pyDetectTypeChecker(tmpDir)
	if strictness != 1 {
		t.Errorf("pyDetectTypeChecker with mypy.ini = %v, want 1", strictness)
	}
}

func TestC2PythonAnalyzer_EmptyTarget(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := newC2PythonAnalyzer(tsParser)

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  "/tmp/empty",
		Files:    nil,
	}

	metrics, err := analyzer.Analyze(target)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	if metrics.LOC != 0 {
		t.Errorf("LOC = %d, want 0 for empty target", metrics.LOC)
	}
}

func TestPyNamingConventions(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"create_user", true},
		{"get_greeting", true},
		{"list_users", true},
		{"createUser", false},    // camelCase is not PEP 8
		{"CreateUser", false},    // PascalCase is not PEP 8 for functions
		{"SOME_CONSTANT", false}, // UPPER_CASE is not snake_case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSnakeCase(tt.name)
			if got != tt.want {
				t.Errorf("isSnakeCase(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPyCamelCase(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"User", true},
		{"ApiResponse", true},
		{"user", false},
		{"api_response", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCamelCase(tt.name)
			if got != tt.want {
				t.Errorf("isCamelCase(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
