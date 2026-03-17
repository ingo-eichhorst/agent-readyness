package c2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

func TestC2TypeScriptAnalyzer_ValidProject(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create Tree-sitter parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := newC2TypeScriptAnalyzer(tsParser)

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

	// TypeAnnotationCoverage: index.ts has zero `any` usages, so coverage should be 100.
	if metrics.TypeAnnotationCoverage != 100 {
		t.Errorf("TypeAnnotationCoverage = %v, want 100 (no `any` usages in index.ts)", metrics.TypeAnnotationCoverage)
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

	analyzer := newC2TypeScriptAnalyzer(tsParser)

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

func TestTSStrictModeMonorepoDetection(t *testing.T) {
	// Simulate a monorepo where the root tsconfig.json has no strict flags
	// but a nested package does (e.g. packages/core/tsconfig.json).
	tmpDir := t.TempDir()

	// Root config: no strict
	rootConfig := `{"compilerOptions": {}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(rootConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Nested package with strict enabled
	pkgDir := filepath.Join(tmpDir, "packages", "core")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedConfig := `{"compilerOptions": {"strict": true}}`
	if err := os.WriteFile(filepath.Join(pkgDir, "tsconfig.json"), []byte(nestedConfig), 0644); err != nil {
		t.Fatal(err)
	}

	isStrict, hasNullChecks := tsDetectStrictMode(tmpDir)
	if !isStrict {
		t.Error("expected strict=true: nested package tsconfig has strict: true")
	}
	if !hasNullChecks {
		t.Error("expected strictNullChecks=true (implied by strict: true in nested config)")
	}

	// Verify that a dir with NO tsconfig files at all returns false
	emptyDir := t.TempDir()
	isStrict, hasNullChecks = tsDetectStrictMode(emptyDir)
	if isStrict {
		t.Error("expected strict=false for dir with no tsconfig files")
	}
	if hasNullChecks {
		t.Error("expected strictNullChecks=false for dir with no tsconfig files")
	}
}

func TestTSAnyUsageCount(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer tsParser.Close()

	// Code with explicit `any` usages
	codeWithAny := []byte(`
function foo(x: any): any { return x; }
const bar: any = 42;
`)
	tree, err := tsParser.ParseFile(types.LangTypeScript, ".ts", codeWithAny)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	defer tree.Close()

	count := tsCountAnyUsages(tree.RootNode(), codeWithAny)
	if count != 3 {
		t.Errorf("tsCountAnyUsages = %d, want 3 (2 in function + 1 in variable)", count)
	}

	// Code with zero `any` usages (using inference)
	codeNoAny := []byte(`
const double = (x: number) => x * 2;
function greet(name: string): string { return "Hello " + name; }
`)
	tree2, err := tsParser.ParseFile(types.LangTypeScript, ".ts", codeNoAny)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	defer tree2.Close()

	count2 := tsCountAnyUsages(tree2.RootNode(), codeNoAny)
	if count2 != 0 {
		t.Errorf("tsCountAnyUsages = %d, want 0 for code without any", count2)
	}
}

func TestTSNamingConsistency(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer tsParser.Close()

	// Well-named code: functions camelCase, classes/interfaces/types PascalCase.
	wellNamed := []byte(`
function createUser(name: string): User { return { name }; }
function parseResponse(data: unknown): ApiResult { return {}; }
class UserService { getData(): void {} }
interface ApiResult { status: number; }
type RequestOptions = { timeout: number; };
`)
	tree, err := tsParser.ParseFile(types.LangTypeScript, ".ts", wellNamed)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	defer tree.Close()

	consistent, total := tsNamingConsistency(tree.RootNode(), wellNamed)
	if total == 0 {
		t.Fatal("total naming-checked identifiers = 0, want > 0")
	}
	if consistent != total {
		t.Errorf("consistent=%d total=%d, want consistent==total for well-named code", consistent, total)
	}
	pct := float64(consistent) / float64(total) * 100
	if pct < 99 {
		t.Errorf("naming consistency = %.1f%%, want 100%% for well-named code", pct)
	}
	t.Logf("Well-named: consistent=%d total=%d (%.1f%%)", consistent, total, pct)

	// Poorly-named code: snake_case functions, lowercase classes.
	badlyNamed := []byte(`
function create_user(name: string): void {}
function parse_response(data: unknown): void {}
class user_service { get_data(): void {} }
interface api_result { status: number; }
type request_options = { timeout: number; };
`)
	tree2, err := tsParser.ParseFile(types.LangTypeScript, ".ts", badlyNamed)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	defer tree2.Close()

	consistent2, total2 := tsNamingConsistency(tree2.RootNode(), badlyNamed)
	if total2 == 0 {
		t.Fatal("total2 = 0, want > 0")
	}
	pct2 := float64(consistent2) / float64(total2) * 100
	if pct2 >= 50 {
		t.Errorf("naming consistency = %.1f%%, want < 50%% for badly-named code", pct2)
	}
	t.Logf("Badly-named: consistent=%d total=%d (%.1f%%)", consistent2, total2, pct2)
}

func TestC2TypeScriptAnalyzer_NamingConsistencySet(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	defer tsParser.Close()

	analyzer := newC2TypeScriptAnalyzer(tsParser)

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
			{
				Path:     filepath.Join(testDir, "src", "utils.ts"),
				RelPath:  "src/utils.ts",
				Language: types.LangTypeScript,
				Class:    types.ClassSource,
			},
		},
	}

	metrics, err := analyzer.Analyze(target)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}

	// NamingConsistency must be computed (non-zero) for the test project
	// which follows TypeScript naming conventions throughout.
	if metrics.NamingConsistency <= 0 {
		t.Errorf("NamingConsistency = %.1f, want > 0 for well-named TypeScript project", metrics.NamingConsistency)
	}
	// The test project follows conventions, so expect high consistency.
	if metrics.NamingConsistency < 80 {
		t.Errorf("NamingConsistency = %.1f, want >= 80 for well-named TypeScript project", metrics.NamingConsistency)
	}
	t.Logf("TypeScript NamingConsistency = %.1f%%, TotalIdentifiers = %d",
		metrics.NamingConsistency, metrics.TotalIdentifiers)
}
