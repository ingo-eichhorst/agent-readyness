package c6

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// testdataDir returns the absolute path to the project testdata directory.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata")
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

func TestPyDetectTests(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")

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

	testFuncs, testFileCount, srcFileCount := pyDetectTests(parsed)

	// Test file count
	if testFileCount != 1 {
		t.Errorf("testFileCount = %d, want 1", testFileCount)
	}
	if srcFileCount != 2 {
		t.Errorf("srcFileCount = %d, want 2", srcFileCount)
	}

	// Test functions: should find test_* functions
	if len(testFuncs) < 3 {
		t.Errorf("expected >= 3 test functions, got %d", len(testFuncs))
	}

	// Check all start with test_
	for _, tf := range testFuncs {
		if tf.Name[:5] != "test_" {
			t.Errorf("test function %q doesn't start with test_", tf.Name)
		}
	}

	t.Logf("Found %d test functions:", len(testFuncs))
	for _, tf := range testFuncs {
		t.Logf("  %s: assertions=%d file=%s line=%d", tf.Name, tf.AssertionCount, tf.File, tf.Line)
	}
}

func TestPyCountAssertions(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  testDir,
		Files: []types.SourceFile{
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

	testFuncs, _, _ := pyDetectTests(parsed)

	funcByName := make(map[string]types.TestFunctionMetric)
	for _, tf := range testFuncs {
		funcByName[tf.Name] = tf
	}

	// test_create_user has 3 assert statements
	if tf, ok := funcByName["test_create_user"]; !ok {
		t.Error("test_create_user not found")
	} else if tf.AssertionCount != 3 {
		t.Errorf("test_create_user assertions = %d, want 3", tf.AssertionCount)
	}

	// test_create_user_with_age has 1 assert statement
	if tf, ok := funcByName["test_create_user_with_age"]; !ok {
		t.Error("test_create_user_with_age not found")
	} else if tf.AssertionCount != 1 {
		t.Errorf("test_create_user_with_age assertions = %d, want 1", tf.AssertionCount)
	}
}

func TestPyAnalyzeIsolation(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  testDir,
		Files: []types.SourceFile{
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

	testFuncs, _, _ := pyDetectTests(parsed)
	isolation := pyAnalyzeIsolation(parsed, testFuncs)

	// test_app.py only imports from app (local), no external deps
	// So isolation should be 100%
	if isolation != 100 {
		t.Errorf("isolation = %.1f, want 100 (no external deps)", isolation)
	}

	t.Logf("Isolation: %.1f%%", isolation)
}

func TestPyC6Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")

	analyzer := NewC6Analyzer(tsParser)
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

	metrics, ok := result.Metrics["c6"].(*types.C6Metrics)
	if !ok {
		t.Fatal("expected C6Metrics in Metrics[\"c6\"]")
	}

	if metrics.TestFileCount <= 0 {
		t.Error("TestFileCount should be > 0")
	}
	if metrics.SourceFileCount <= 0 {
		t.Error("SourceFileCount should be > 0")
	}
	if metrics.TestToCodeRatio <= 0 {
		t.Error("TestToCodeRatio should be > 0")
	}
	if len(metrics.TestFunctions) == 0 {
		t.Error("TestFunctions should not be empty")
	}

	t.Logf("C6 Python: TestFiles=%d SrcFiles=%d TestToCode=%.2f Isolation=%.1f TestFuncs=%d",
		metrics.TestFileCount, metrics.SourceFileCount, metrics.TestToCodeRatio,
		metrics.TestIsolation, len(metrics.TestFunctions))
}

// TestC6_GoRegressionWithNewConstructor verifies Go C6 analysis still works.
func TestC6_GoRegressionWithNewConstructor(t *testing.T) {
	pkgs := loadTestPackages(t, "valid-go-project")

	analyzer := NewC6Analyzer(nil)
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics, ok := result.Metrics["c6"].(*types.C6Metrics)
	if !ok {
		t.Fatal("expected C6Metrics")
	}

	t.Logf("Go C6 regression: TestFiles=%d SrcFiles=%d TestToCode=%.2f",
		metrics.TestFileCount, metrics.SourceFileCount, metrics.TestToCodeRatio)
}
