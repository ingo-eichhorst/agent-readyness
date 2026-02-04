package c2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

func TestC2TypeScriptAnalyzer_ValidProject(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := NewC2TypeScriptAnalyzer(tsParser)

	testDir, err := filepath.Abs("../../../testdata/valid-ts-project")
	if err != nil {
		t.Fatalf("cannot resolve testdata path: %v", err)
	}

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

	metrics, err := analyzer.Analyze(target)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// Type annotation coverage should be > 0 (index.ts has full type annotations)
	if metrics.TypeAnnotationCoverage <= 0 {
		t.Errorf("TypeAnnotationCoverage = %v, want > 0", metrics.TypeAnnotationCoverage)
	}

	// TotalFunctions should be > 0
	if metrics.TotalFunctions <= 0 {
		t.Errorf("TotalFunctions = %d, want > 0", metrics.TotalFunctions)
	}

	// LOC should be > 0
	if metrics.LOC <= 0 {
		t.Errorf("LOC = %d, want > 0", metrics.LOC)
	}

	// TypeStrictness should be 1 (tsconfig.json has strict: true)
	if metrics.TypeStrictness != 1 {
		t.Errorf("TypeStrictness = %v, want 1 (strict mode enabled in tsconfig.json)", metrics.TypeStrictness)
	}

	// Magic number ratio should be >= 0
	if metrics.MagicNumberRatio < 0 {
		t.Errorf("MagicNumberRatio = %v, want >= 0", metrics.MagicNumberRatio)
	}

	t.Logf("TypeScript C2 metrics: TypeAnnotation=%.1f MagicNumberRatio=%.2f TypeStrictness=%.0f NullSafety=%.1f Functions=%d MagicNumbers=%d LOC=%d",
		metrics.TypeAnnotationCoverage, metrics.MagicNumberRatio,
		metrics.TypeStrictness, metrics.NullSafety,
		metrics.TotalFunctions, metrics.MagicNumberCount, metrics.LOC)
}

func TestC2TypeScriptAnalyzer_StrictModeDetection(t *testing.T) {
	testDir, _ := filepath.Abs("../../../testdata/valid-ts-project")
	isStrict, hasNullChecks := tsDetectStrictMode(testDir)
	if !isStrict {
		t.Error("tsDetectStrictMode: expected strict=true (tsconfig has strict: true)")
	}
	if !hasNullChecks {
		t.Error("tsDetectStrictMode: expected strictNullChecks=true (implied by strict: true)")
	}

	// Test with non-existent directory
	isStrict, hasNullChecks = tsDetectStrictMode("/tmp/nonexistent")
	if isStrict {
		t.Error("tsDetectStrictMode: expected false for non-existent dir")
	}
	if hasNullChecks {
		t.Error("tsDetectStrictMode: expected false for non-existent dir")
	}
}

func TestC2TypeScriptAnalyzer_EmptyTarget(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := NewC2TypeScriptAnalyzer(tsParser)

	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
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

func TestC2Analyzer_MultiLanguageDispatch(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := NewC2Analyzer(tsParser)

	pyDir, _ := filepath.Abs("../../../testdata/valid-python-project")
	tsDir, _ := filepath.Abs("../../../testdata/valid-ts-project")

	targets := []*types.AnalysisTarget{
		{
			Language: types.LangPython,
			RootDir:  pyDir,
			Files: []types.SourceFile{
				{
					Path:     filepath.Join(pyDir, "app.py"),
					RelPath:  "app.py",
					Language: types.LangPython,
					Class:    types.ClassSource,
				},
			},
		},
		{
			Language: types.LangTypeScript,
			RootDir:  tsDir,
			Files: []types.SourceFile{
				{
					Path:     filepath.Join(tsDir, "src", "index.ts"),
					RelPath:  "src/index.ts",
					Language: types.LangTypeScript,
					Class:    types.ClassSource,
				},
			},
		},
	}

	result, err := analyzer.Analyze(targets)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	c2 := result.Metrics["c2"].(*types.C2Metrics)

	// Should have both Python and TypeScript in PerLanguage
	if _, ok := c2.PerLanguage[types.LangPython]; !ok {
		t.Error("missing Python in PerLanguage")
	}
	if _, ok := c2.PerLanguage[types.LangTypeScript]; !ok {
		t.Error("missing TypeScript in PerLanguage")
	}

	// Aggregate should be non-nil
	if c2.Aggregate == nil {
		t.Fatal("Aggregate is nil for multi-language targets")
	}

	// Aggregate LOC should be sum of both
	pyLOC := c2.PerLanguage[types.LangPython].LOC
	tsLOC := c2.PerLanguage[types.LangTypeScript].LOC
	if c2.Aggregate.LOC != pyLOC+tsLOC {
		t.Errorf("Aggregate.LOC = %d, want %d (sum of Python %d + TypeScript %d)",
			c2.Aggregate.LOC, pyLOC+tsLOC, pyLOC, tsLOC)
	}

	t.Logf("Multi-language C2: Python LOC=%d, TypeScript LOC=%d, Aggregate LOC=%d",
		pyLOC, tsLOC, c2.Aggregate.LOC)
}

func TestTSStrictModeWithCustomConfig(t *testing.T) {
	// Create temp dir with custom tsconfig.json
	tmpDir := t.TempDir()

	// Test with individual strict flags
	config := `{
		"compilerOptions": {
			"strictNullChecks": true,
			"noImplicitAny": true,
			"strictFunctionTypes": true
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	isStrict, hasNullChecks := tsDetectStrictMode(tmpDir)
	if !isStrict {
		t.Error("expected strict=true with all individual flags")
	}
	if !hasNullChecks {
		t.Error("expected strictNullChecks=true")
	}

	// Test with strict: false and no individual flags
	config = `{
		"compilerOptions": {
			"strict": false
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	isStrict, hasNullChecks = tsDetectStrictMode(tmpDir)
	if isStrict {
		t.Error("expected strict=false with strict: false")
	}
	if hasNullChecks {
		t.Error("expected strictNullChecks=false with strict: false")
	}
}
