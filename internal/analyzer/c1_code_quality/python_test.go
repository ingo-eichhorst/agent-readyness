package c1

import (
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestPyAnalyzeFunctions_Complexity(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, err := filepath.Abs("../../../testdata/valid-python-project")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "utils.py"),
				RelPath:  "utils.py",
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

	functions := pyAnalyzeFunctions(parsed)
	if len(functions) == 0 {
		t.Fatal("expected at least one function, got none")
	}

	funcByName := make(map[string]types.FunctionMetric)
	for _, f := range functions {
		funcByName[f.Name] = f
	}

	// simple_add should have complexity 1 (no branches)
	if f, ok := funcByName["simple_add"]; !ok {
		t.Error("simple_add not found")
	} else if f.Complexity != 1 {
		t.Errorf("simple_add complexity = %d, want 1", f.Complexity)
	}

	// process_record has multiple branches: if/elif/elif/else (3 if-like) + nested ifs + for + while + boolean_operator
	pr, ok := funcByName["DataProcessor.process_record"]
	if !ok {
		t.Fatal("DataProcessor.process_record not found")
	}
	if pr.Complexity <= 5 {
		t.Errorf("DataProcessor.process_record complexity = %d, want > 5", pr.Complexity)
	}
	t.Logf("DataProcessor.process_record complexity = %d", pr.Complexity)

	// validate should have complexity 1 (just a return with comprehension)
	if f, ok := funcByName["DataProcessor.validate"]; !ok {
		t.Error("DataProcessor.validate not found")
	} else if f.Complexity != 1 {
		t.Errorf("DataProcessor.validate complexity = %d, want 1", f.Complexity)
	}

	// find_duplicates should have complexity 2 (for + if)
	if f, ok := funcByName["find_duplicates"]; !ok {
		t.Error("find_duplicates not found")
	} else if f.Complexity != 3 {
		t.Errorf("find_duplicates complexity = %d, want 3 (for + if + else)", f.Complexity)
	}

	t.Logf("Found %d functions:", len(functions))
	for _, f := range functions {
		t.Logf("  %s: complexity=%d lines=%d", f.Name, f.Complexity, f.LineCount)
	}
}

func TestPyAnalyzeFunctions_LineCount(t *testing.T) {
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
				Path:     filepath.Join(testDir, "utils.py"),
				RelPath:  "utils.py",
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

	functions := pyAnalyzeFunctions(parsed)

	funcByName := make(map[string]types.FunctionMetric)
	for _, f := range functions {
		funcByName[f.Name] = f
	}

	// simple_add should be a small function (about 3 lines)
	if f, ok := funcByName["simple_add"]; !ok {
		t.Error("simple_add not found")
	} else if f.LineCount < 2 {
		t.Errorf("simple_add line count = %d, want >= 2", f.LineCount)
	}

	// process_record should be a large function
	if f, ok := funcByName["DataProcessor.process_record"]; !ok {
		t.Error("DataProcessor.process_record not found")
	} else if f.LineCount < 15 {
		t.Errorf("DataProcessor.process_record line count = %d, want >= 15", f.LineCount)
	}
}

func TestPyAnalyzeFileSizes(t *testing.T) {
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
				Path:     filepath.Join(testDir, "utils.py"),
				RelPath:  "utils.py",
				Language: types.LangPython,
				Class:    types.ClassSource,
			},
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

	summary := pyAnalyzeFileSizes(parsed)

	if summary.Max <= 0 {
		t.Error("FileSize.Max should be > 0")
	}
	if summary.Avg <= 0 {
		t.Error("FileSize.Avg should be > 0")
	}
	if summary.MaxEntity == "" {
		t.Error("FileSize.MaxEntity should not be empty")
	}

	t.Logf("FileSize: avg=%.1f max=%d maxEntity=%s", summary.Avg, summary.Max, summary.MaxEntity)
}

func TestPyC1Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-python-project")

	analyzer := NewC1Analyzer(tsParser)
	targets := []*types.AnalysisTarget{
		{
			Language: types.LangPython,
			RootDir:  testDir,
			Files: []types.SourceFile{
				{
					Path:     filepath.Join(testDir, "utils.py"),
					RelPath:  "utils.py",
					Language: types.LangPython,
					Class:    types.ClassSource,
				},
				{
					Path:     filepath.Join(testDir, "app.py"),
					RelPath:  "app.py",
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

	metrics, ok := result.Metrics["c1"].(*C1MetricsResult)
	if !ok {
		t.Fatal("expected C1MetricsResult in Metrics[\"c1\"]")
	}

	if metrics.CyclomaticComplexity.Avg <= 0 {
		t.Error("CyclomaticComplexity.Avg should be > 0")
	}
	if metrics.FunctionLength.Avg <= 0 {
		t.Error("FunctionLength.Avg should be > 0")
	}
	if metrics.FileSize.Max <= 0 {
		t.Error("FileSize.Max should be > 0")
	}

	t.Logf("C1 Python: ComplexityAvg=%.1f FuncLenAvg=%.1f FileSizeMax=%d Functions=%d",
		metrics.CyclomaticComplexity.Avg, metrics.FunctionLength.Avg,
		metrics.FileSize.Max, len(metrics.Functions))
}

func TestPyIsTestFileByPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"test_app.py", true},
		{"app_test.py", true},
		{"conftest.py", true},
		{"app.py", false},
		{"utils.py", false},
		{"subdir/test_foo.py", true},
		{"subdir/foo.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTestFileByPath(tt.path)
			if got != tt.want {
				t.Errorf("isTestFileByPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestC1_GoRegressionWithNewConstructor verifies Go C1 analysis still works
// with the new NewC1Analyzer constructor.
func TestC1_GoRegressionWithNewConstructor(t *testing.T) {
	pkgs := loadTestPackages(t, "complexity")

	analyzer := NewC1Analyzer(nil) // nil tsParser is fine for Go-only
	analyzer.SetGoPackages(pkgs)
	result, err := analyzer.Analyze(nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metrics, ok := result.Metrics["c1"].(*C1MetricsResult)
	if !ok {
		t.Fatal("expected C1MetricsResult")
	}

	funcComplexity := make(map[string]int)
	for _, fm := range metrics.Functions {
		funcComplexity[fm.Name] = fm.Complexity
	}

	if c, ok := funcComplexity["SimpleFunc"]; !ok || c != 1 {
		t.Errorf("SimpleFunc complexity = %d, want 1", c)
	}
	if c, ok := funcComplexity["MultiBranch"]; !ok || c < 6 {
		t.Errorf("MultiBranch complexity = %d, want >= 6", c)
	}
}
