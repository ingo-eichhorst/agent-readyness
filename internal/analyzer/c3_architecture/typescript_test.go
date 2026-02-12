package c3

import (
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestTsBuildImportGraph(t *testing.T) {
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
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	graph := tsBuildImportGraph(parsed)

	// utils.ts imports from ./index: "import { User } from './index'"
	utilsKey := tsNormalizePath("src/utils.ts")
	indexKey := tsNormalizePath("src/index.ts")

	utilsImports := graph.Forward[utilsKey]
	found := false
	for _, imp := range utilsImports {
		if imp == indexKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected utils to import index, got Forward[%s]=%v", utilsKey, utilsImports)
	}

	// index.test.ts imports from ./index
	testKey := tsNormalizePath("src/index.test.ts")
	testImports := graph.Forward[testKey]
	found = false
	for _, imp := range testImports {
		if imp == indexKey {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected index.test to import index, got Forward[%s]=%v", testKey, testImports)
	}

	t.Logf("Import graph Forward: %v", graph.Forward)
	t.Logf("Import graph Reverse: %v", graph.Reverse)
}

func TestTsDetectDeadCode(t *testing.T) {
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
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	dead := tsDetectDeadCode(parsed)

	if len(dead) == 0 {
		t.Error("expected at least one dead export from utils.ts")
	}

	deadNames := make(map[string]bool)
	for _, d := range dead {
		deadNames[d.Name] = true
		t.Logf("Dead export: %s (%s) in %s:%d", d.Name, d.Kind, d.File, d.Line)
	}

	// unusedHelper and UnusedProcessor should be flagged as dead exports
	if !deadNames["unusedHelper"] {
		t.Error("expected unusedHelper to be flagged as dead export")
	}
	if !deadNames["UnusedProcessor"] {
		t.Error("expected UnusedProcessor to be flagged as dead export")
	}
}

func TestTsAnalyzeDirectoryDepth(t *testing.T) {
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
		},
	}

	parsed, err := tsParser.ParseTargetFiles(target)
	if err != nil {
		t.Fatalf("ParseTargetFiles error: %v", err)
	}
	defer parser.CloseAll(parsed)

	maxDepth, avgDepth := tsAnalyzeDirectoryDepth(parsed, testDir)

	// Files are in src/ dir, so depth should be 1
	if maxDepth != 1 {
		t.Errorf("maxDepth = %d, want 1 for src/ level files", maxDepth)
	}
	if avgDepth != 1 {
		t.Errorf("avgDepth = %f, want 1 for src/ level files", avgDepth)
	}
}

func TestTsNormalizePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"src/index.ts", "src"},       // index files normalize to directory
		{"src/utils.tsx", "src/utils"},
		{"lib/helper.js", "lib/helper"},
		{"src/index", "src"},          // bare index also normalizes
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := tsNormalizePath(tt.path)
			if got != tt.want {
				t.Errorf("tsNormalizePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestTsC3Integration(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	a := NewC3Analyzer(tsParser)
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

	result, err := a.Analyze(targets)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	metrics, ok := result.Metrics["c3"].(*types.C3Metrics)
	if !ok {
		t.Fatal("expected C3Metrics in Metrics[\"c3\"]")
	}

	if metrics.MaxDirectoryDepth <= 0 {
		t.Error("MaxDirectoryDepth should be > 0 for files in src/")
	}

	t.Logf("C3 TypeScript: MaxDepth=%d AvgDepth=%.1f DeadExports=%d CircularDeps=%d Fanout=%.1f",
		metrics.MaxDirectoryDepth, metrics.AvgDirectoryDepth,
		len(metrics.DeadExports), len(metrics.CircularDeps), metrics.ModuleFanout.Avg)
}
