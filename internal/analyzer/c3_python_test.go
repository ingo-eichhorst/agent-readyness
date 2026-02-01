package analyzer

import (
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestPyBuildImportGraph(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../testdata/valid-python-project")

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
			{
				Path:     filepath.Join(testDir, "utils.py"),
				RelPath:  "utils.py",
				Language: types.LangPython,
				Class:    types.ClassSource,
			},
			{
				Path:     filepath.Join(testDir, "test_app.py"),
				RelPath:  "test_app.py",
				Language: types.LangPython,
				Class:    types.ClassTest,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	graph := pyBuildImportGraph(parsed)

	// test_app.py imports from app.py: "from app import User, create_user, get_greeting"
	testAppImports := graph.Forward["test_app"]
	found := false
	for _, imp := range testAppImports {
		if imp == "app" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test_app to import app, got Forward[test_app]=%v", testAppImports)
	}

	// Reverse: app should be imported by test_app
	appImporters := graph.Reverse["app"]
	found = false
	for _, imp := range appImporters {
		if imp == "test_app" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected app to be imported by test_app, got Reverse[app]=%v", appImporters)
	}

	t.Logf("Import graph Forward: %v", graph.Forward)
	t.Logf("Import graph Reverse: %v", graph.Reverse)
}

func TestPyDetectDeadCode(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../testdata/valid-python-project")

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
			{
				Path:     filepath.Join(testDir, "utils.py"),
				RelPath:  "utils.py",
				Language: types.LangPython,
				Class:    types.ClassSource,
			},
			{
				Path:     filepath.Join(testDir, "test_app.py"),
				RelPath:  "test_app.py",
				Language: types.LangPython,
				Class:    types.ClassTest,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	dead := pyDetectDeadCode(parsed)

	// utils.py has functions not imported anywhere (DataProcessor, simple_add, etc.)
	if len(dead) == 0 {
		t.Error("expected at least one dead export from utils.py")
	}

	deadNames := make(map[string]bool)
	for _, d := range dead {
		deadNames[d.Name] = true
		t.Logf("Dead export: %s (%s) in %s:%d", d.Name, d.Kind, d.File, d.Line)
	}

	// DataProcessor should be flagged (not imported by other source files)
	if !deadNames["DataProcessor"] {
		t.Error("expected DataProcessor to be flagged as dead export")
	}
}

func TestPyAnalyzeDirectoryDepth(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../testdata/valid-python-project")

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

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	maxDepth, avgDepth := pyAnalyzeDirectoryDepth(parsed, testDir)

	// Files are in root dir, so depth should be 0
	if maxDepth != 0 {
		t.Errorf("maxDepth = %d, want 0 for root-level files", maxDepth)
	}
	if avgDepth != 0 {
		t.Errorf("avgDepth = %f, want 0 for root-level files", avgDepth)
	}
}

func TestPyFileToModule(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"app.py", "app"},
		{"utils.py", "utils"},
		{"pkg/sub/foo.py", "pkg.sub.foo"},
		{"pkg/__init__.py", "pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pyFileToModule(tt.path)
			if got != tt.want {
				t.Errorf("pyFileToModule(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestPyC3Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../testdata/valid-python-project")

	analyzer := NewC3Analyzer(tsParser)
	targets := []*types.AnalysisTarget{
		{
			Language: types.LangPython,
			RootDir:  testDir,
			Files: []types.SourceFile{
				{
					Path:     filepath.Join(testDir, "app.py"),
					RelPath:  "app.py",
					Language: types.LangPython,
					Class:    types.ClassSource,
				},
				{
					Path:     filepath.Join(testDir, "utils.py"),
					RelPath:  "utils.py",
					Language: types.LangPython,
					Class:    types.ClassSource,
				},
				{
					Path:     filepath.Join(testDir, "test_app.py"),
					RelPath:  "test_app.py",
					Language: types.LangPython,
					Class:    types.ClassTest,
				},
			},
		},
	}

	result, err := analyzer.Analyze(targets)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	metrics, ok := result.Metrics["c3"].(*types.C3Metrics)
	if !ok {
		t.Fatal("expected C3Metrics in Metrics[\"c3\"]")
	}

	// With flat directory, maxDepth is 0 but that's correct
	t.Logf("C3 Python: MaxDepth=%d AvgDepth=%.1f DeadExports=%d CircularDeps=%d",
		metrics.MaxDirectoryDepth, metrics.AvgDirectoryDepth,
		len(metrics.DeadExports), len(metrics.CircularDeps))
}

// TestC3_GoRegressionWithNewConstructor verifies Go C3 analysis still works.
func TestC3_GoRegressionWithNewConstructor(t *testing.T) {
	pkgs := loadTestPackages(t, "coupling")

	analyzer := NewC3Analyzer(nil)
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics, ok := result.Metrics["c3"].(*types.C3Metrics)
	if !ok {
		t.Fatal("expected C3Metrics")
	}

	// coupling testdata should have some depth and dead exports
	t.Logf("Go C3 regression: MaxDepth=%d AvgDepth=%.1f DeadExports=%d",
		metrics.MaxDirectoryDepth, metrics.AvgDirectoryDepth, len(metrics.DeadExports))
}
