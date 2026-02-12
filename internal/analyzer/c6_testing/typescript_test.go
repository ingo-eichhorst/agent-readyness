package c6

import (
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestTsDetectTests(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "src", "index.ts"),
				RelPath:  "src/index.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassSource,
			},
			{
				Path:     filepath.Join(testDir, "src", "utils.ts"),
				RelPath:  "src/utils.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassSource,
			},
			{
				Path:     filepath.Join(testDir, "src", "index.test.ts"),
				RelPath:  "src/index.test.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassTest,
			},
			{
				Path:     filepath.Join(testDir, "src", "app.test.ts"),
				RelPath:  "src/app.test.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassTest,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	testFuncs, testFileCount, srcFileCount := tsDetectTests(parsed)

	// Test file count: index.test.ts + app.test.ts
	if testFileCount != 2 {
		t.Errorf("testFileCount = %d, want 2", testFileCount)
	}
	if srcFileCount != 2 {
		t.Errorf("srcFileCount = %d, want 2", srcFileCount)
	}

	// Test functions: should find it() and test() calls
	if len(testFuncs) < 3 {
		t.Errorf("expected >= 3 test functions, got %d", len(testFuncs))
	}

	t.Logf("Found %d test functions:", len(testFuncs))
	for _, tf := range testFuncs {
		t.Logf("  %s: assertions=%d file=%s line=%d", tf.Name, tf.AssertionCount, tf.File, tf.Line)
	}
}

func TestTsCountAssertions(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "src", "app.test.ts"),
				RelPath:  "src/app.test.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassTest,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	testFuncs, _, _ := tsDetectTests(parsed)

	funcByName := make(map[string]types.TestFunctionMetric)
	for _, tf := range testFuncs {
		funcByName[tf.Name] = tf
	}

	// "should add two numbers" has 3 expect calls
	if tf, ok := funcByName["should add two numbers"]; !ok {
		t.Error("'should add two numbers' not found")
	} else if tf.AssertionCount != 3 {
		t.Errorf("'should add two numbers' assertions = %d, want 3", tf.AssertionCount)
	}

	// "should handle negative numbers" has 1 expect call
	if tf, ok := funcByName["should handle negative numbers"]; !ok {
		t.Error("'should handle negative numbers' not found")
	} else if tf.AssertionCount != 1 {
		t.Errorf("'should handle negative numbers' assertions = %d, want 1", tf.AssertionCount)
	}
}

func TestTsAnalyzeIsolation(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "src", "app.test.ts"),
				RelPath:  "src/app.test.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassTest,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	testFuncs, _, _ := tsDetectTests(parsed)
	isolation := tsAnalyzeIsolation(parsed, testFuncs)

	// app.test.ts only imports from ./utils (local), no external deps
	if isolation != 100 {
		t.Errorf("isolation = %.1f, want 100 (no external deps)", isolation)
	}

	t.Logf("Isolation: %.1f%%", isolation)
}

func TestTsC6Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	analyzer := NewC6Analyzer(tsParser)
	targets := []*types.AnalysisTarget{
		{
			Language: types.LangTypeScript,
			RootDir:  testDir,
			Files: []types.SourceFile{
				{
					Path:     filepath.Join(testDir, "src", "index.ts"),
					RelPath:  "src/index.ts",
					Language: types.LangTypeScript,
					Class:    types.ClassSource,
				},
				{
					Path:     filepath.Join(testDir, "src", "utils.ts"),
					RelPath:  "src/utils.ts",
					Language: types.LangTypeScript,
					Class:    types.ClassSource,
				},
				{
					Path:     filepath.Join(testDir, "src", "index.test.ts"),
					RelPath:  "src/index.test.ts",
					Language: types.LangTypeScript,
					Class:    types.ClassTest,
				},
				{
					Path:     filepath.Join(testDir, "src", "app.test.ts"),
					RelPath:  "src/app.test.ts",
					Language: types.LangTypeScript,
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

	t.Logf("C6 TypeScript: TestFiles=%d SrcFiles=%d TestToCode=%.2f Isolation=%.1f TestFuncs=%d AssertDensity=%.1f",
		metrics.TestFileCount, metrics.SourceFileCount, metrics.TestToCodeRatio,
		metrics.TestIsolation, len(metrics.TestFunctions), metrics.AssertionDensity.Avg)
}
