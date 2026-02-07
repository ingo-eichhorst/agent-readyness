package c1

import (
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestTsAnalyzeFunctions_Complexity(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, err := filepath.Abs("../../../testdata/valid-ts-project")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  testDir,
		Files: []types.SourceFile{
			{
				Path:     filepath.Join(testDir, "src", "utils.ts"),
				RelPath:  "src/utils.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassSource,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	functions := tsAnalyzeFunctions(parsed)
	if len(functions) == 0 {
		t.Fatal("expected at least one function, got none")
	}

	funcByName := make(map[string]types.FunctionMetric)
	for _, f := range functions {
		funcByName[f.Name] = f
	}

	// add should have complexity 1 (no branches)
	if f, ok := funcByName["add"]; !ok {
		t.Error("add not found")
	} else if f.Complexity != 1 {
		t.Errorf("add complexity = %d, want 1", f.Complexity)
	}

	// processData has multiple branches: for, if, if, else if, if, ||, else, switch with 3 cases
	pd, ok := funcByName["processData"]
	if !ok {
		t.Fatal("processData not found")
	}
	if pd.Complexity <= 5 {
		t.Errorf("processData complexity = %d, want > 5", pd.Complexity)
	}
	t.Logf("processData complexity = %d", pd.Complexity)

	// multiply (arrow function) should have complexity 1
	if f, ok := funcByName["multiply"]; !ok {
		t.Error("multiply not found")
	} else if f.Complexity != 1 {
		t.Errorf("multiply complexity = %d, want 1", f.Complexity)
	}

	// categorize (arrow function with branches) should have complexity > 1
	if f, ok := funcByName["categorize"]; !ok {
		t.Error("categorize not found")
	} else if f.Complexity <= 1 {
		t.Errorf("categorize complexity = %d, want > 1", f.Complexity)
	}

	// DataProcessor.addItem should have complexity 2 (if with &&)
	if f, ok := funcByName["DataProcessor.addItem"]; !ok {
		t.Error("DataProcessor.addItem not found")
	} else if f.Complexity < 2 {
		t.Errorf("DataProcessor.addItem complexity = %d, want >= 2", f.Complexity)
	}

	// DataProcessor.process should have complexity >= 2 (for + if)
	if f, ok := funcByName["DataProcessor.process"]; !ok {
		t.Error("DataProcessor.process not found")
	} else if f.Complexity < 2 {
		t.Errorf("DataProcessor.process complexity = %d, want >= 2", f.Complexity)
	}

	t.Logf("Found %d functions:", len(functions))
	for _, f := range functions {
		t.Logf("  %s: complexity=%d lines=%d", f.Name, f.Complexity, f.LineCount)
	}
}

func TestTsAnalyzeFunctions_LineCount(t *testing.T) {
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
				Path:     filepath.Join(testDir, "src", "utils.ts"),
				RelPath:  "src/utils.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassSource,
			},
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	functions := tsAnalyzeFunctions(parsed)

	funcByName := make(map[string]types.FunctionMetric)
	for _, f := range functions {
		funcByName[f.Name] = f
	}

	// add should be a small function (about 3 lines)
	if f, ok := funcByName["add"]; !ok {
		t.Error("add not found")
	} else if f.LineCount < 2 {
		t.Errorf("add line count = %d, want >= 2", f.LineCount)
	}

	// processData should be a large function
	if f, ok := funcByName["processData"]; !ok {
		t.Error("processData not found")
	} else if f.LineCount < 15 {
		t.Errorf("processData line count = %d, want >= 15", f.LineCount)
	}
}

func TestTsAnalyzeFileSizes(t *testing.T) {
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
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	summary := tsAnalyzeFileSizes(parsed)

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

func TestTsC1Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	analyzer := NewC1Analyzer(tsParser)
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

	t.Logf("C1 TypeScript: ComplexityAvg=%.1f FuncLenAvg=%.1f FileSizeMax=%d Functions=%d",
		metrics.CyclomaticComplexity.Avg, metrics.FunctionLength.Avg,
		metrics.FileSize.Max, len(metrics.Functions))
}

func TestTsIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"src/app.test.ts", true},
		{"src/app.spec.ts", true},
		{"src/App.test.tsx", true},
		{"src/App.spec.tsx", true},
		{"__tests__/app.ts", true},
		{"src/index.ts", false},
		{"src/utils.ts", false},
		{"lib/helper.tsx", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := shared.TsIsTestFile(tt.path)
			if got != tt.want {
				t.Errorf("shared.TsIsTestFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
