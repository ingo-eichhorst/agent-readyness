package c4

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// findProjectRoot walks up from cwd to locate the .git directory.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("no .git directory found in parent chain")
		}
		dir = parent
	}
}

// isVendorPath checks if a path is in vendor directory.
func isVendorPath(path string) bool {
	return strings.Contains(path, "/vendor/") || strings.HasPrefix(path, "vendor/")
}

// isTestFile checks if a path is a test file.
func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

func TestC4Analyzer_Name(t *testing.T) {
	a := NewC4Analyzer(nil)
	if got := a.Name(); got != "C4: Documentation Quality" {
		t.Errorf("Name() = %q, want %q", got, "C4: Documentation Quality")
	}
}

func TestC4Analyzer_EmptyTargets(t *testing.T) {
	a := NewC4Analyzer(nil)

	_, err := a.Analyze(nil)
	if err == nil {
		t.Error("Analyze(nil) should return error")
	}

	_, err = a.Analyze([]*types.AnalysisTarget{})
	if err == nil {
		t.Error("Analyze([]) should return error")
	}
}

func TestAnalyzeReadme(t *testing.T) {
	dir := t.TempDir()

	// Test missing README
	present, wordCount := analyzeReadme(dir)
	if present {
		t.Error("expected present=false when README is missing")
	}
	if wordCount != 0 {
		t.Errorf("expected wordCount=0, got %d", wordCount)
	}

	// Create README.md with known content
	readmeContent := "Hello World\n\nThis is a test README with some words."
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	present, wordCount = analyzeReadme(dir)
	if !present {
		t.Error("expected present=true when README.md exists")
	}
	// "Hello World This is a test README with some words" = 10 words
	if wordCount != 10 {
		t.Errorf("expected wordCount=10, got %d", wordCount)
	}
}

func TestAnalyzeReadme_Missing(t *testing.T) {
	dir := t.TempDir()

	present, wordCount := analyzeReadme(dir)
	if present {
		t.Error("expected present=false when no README exists")
	}
	if wordCount != 0 {
		t.Errorf("expected wordCount=0, got %d", wordCount)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"hello", 1},
		{"hello world", 2},
		{"hello  world", 2}, // multiple spaces
		{"hello\nworld", 2}, // newline
		{"hello\t\nworld", 2},
		{"The quick brown fox jumps over the lazy dog", 9},
	}

	for _, tc := range tests {
		got := countWords(tc.text)
		if got != tc.want {
			t.Errorf("countWords(%q) = %d, want %d", tc.text, got, tc.want)
		}
	}
}

func TestAnalyzeChangelog(t *testing.T) {
	dir := t.TempDir()

	// Test missing CHANGELOG
	if analyzeChangelog(dir) {
		t.Error("expected false when CHANGELOG is missing")
	}

	// Create CHANGELOG.md
	if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte("# Changelog"), 0644); err != nil {
		t.Fatal(err)
	}

	if !analyzeChangelog(dir) {
		t.Error("expected true when CHANGELOG.md exists")
	}
}

func TestAnalyzeExamples(t *testing.T) {
	dir := t.TempDir()

	// Test missing examples
	if analyzeExamples(dir) {
		t.Error("expected false when examples are missing")
	}

	// Create examples/ directory
	if err := os.Mkdir(filepath.Join(dir, "examples"), 0755); err != nil {
		t.Fatal(err)
	}

	if !analyzeExamples(dir) {
		t.Error("expected true when examples/ directory exists")
	}
}

func TestAnalyzeExamples_CodeBlocks(t *testing.T) {
	dir := t.TempDir()

	// Create README.md with code blocks
	readmeContent := `# Example

Here is some code:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatal(err)
	}

	if !analyzeExamples(dir) {
		t.Error("expected true when README.md has code blocks")
	}
}

func TestAnalyzeContributing(t *testing.T) {
	dir := t.TempDir()

	// Test missing CONTRIBUTING
	if analyzeContributing(dir) {
		t.Error("expected false when CONTRIBUTING is missing")
	}

	// Create CONTRIBUTING.md
	if err := os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"), []byte("# Contributing"), 0644); err != nil {
		t.Fatal(err)
	}

	if !analyzeContributing(dir) {
		t.Error("expected true when CONTRIBUTING.md exists")
	}
}

func TestAnalyzeDiagrams(t *testing.T) {
	dir := t.TempDir()

	// Test missing diagrams
	if analyzeDiagrams(dir) {
		t.Error("expected false when no diagrams exist")
	}

	// Create docs/ directory with architecture diagram
	docsDir := filepath.Join(dir, "docs")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "architecture-diagram.png"), []byte("PNG"), 0644); err != nil {
		t.Fatal(err)
	}

	if !analyzeDiagrams(dir) {
		t.Error("expected true when docs/architecture-diagram.png exists")
	}
}

func TestFindGitRoot(t *testing.T) {
	// Layout: parent/.git/  parent/sub/nested/
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(parent, "sub", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	if got := findGitRoot(nested); got != parent {
		t.Errorf("findGitRoot(nested) = %q, want %q", got, parent)
	}
	if got := findGitRoot(parent); got != parent {
		t.Errorf("findGitRoot(parent) = %q, want %q", got, parent)
	}

	// .git as a file (git worktree style).
	wtParent := t.TempDir()
	if err := os.WriteFile(filepath.Join(wtParent, ".git"), []byte("gitdir: /tmp/x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := findGitRoot(wtParent); got != wtParent {
		t.Errorf("findGitRoot(.git as file) = %q, want %q", got, wtParent)
	}
}

func TestAnalyzeStaticDocs_GitRootFallback(t *testing.T) {
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"README.md", "CHANGELOG.md", "CONTRIBUTING.md"} {
		if err := os.WriteFile(filepath.Join(parent, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(parent, "examples"), 0755); err != nil {
		t.Fatal(err)
	}
	docs := filepath.Join(parent, "docs")
	if err := os.Mkdir(docs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "architecture-diagram.png"), []byte("PNG"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(parent, "core")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	metrics := &types.C4Metrics{}
	analyzeStaticDocs(sub, metrics)

	if !metrics.ReadmePresent {
		t.Error("ReadmePresent: want true (root README should be detected)")
	}
	if metrics.ReadmeWordCount == 0 {
		t.Error("ReadmeWordCount: want > 0 when root README is found")
	}
	if !metrics.ChangelogPresent {
		t.Error("ChangelogPresent: want true")
	}
	if !metrics.ContributingPresent {
		t.Error("ContributingPresent: want true")
	}
	if !metrics.ExamplesPresent {
		t.Error("ExamplesPresent: want true (examples/ at root)")
	}
	if !metrics.DiagramsPresent {
		t.Error("DiagramsPresent: want true (docs/architecture-diagram.png at root)")
	}
}

func TestAnalyzeStaticDocs_ScanDirTakesPrecedence(t *testing.T) {
	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "README.md"), []byte("root readme one two three"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(parent, "core")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	subReadme := "scan dir readme with notably more words than the root readme has"
	if err := os.WriteFile(filepath.Join(sub, "README.md"), []byte(subReadme), 0644); err != nil {
		t.Fatal(err)
	}

	metrics := &types.C4Metrics{}
	analyzeStaticDocs(sub, metrics)

	wantWords := countWords(subReadme)
	if metrics.ReadmeWordCount != wantWords {
		t.Errorf("ReadmeWordCount = %d, want %d (scan-dir README must win over git-root README)", metrics.ReadmeWordCount, wantWords)
	}
}

func TestAnalyzeStaticDocs_NoFallbackOutsideRepo(t *testing.T) {
	// Create a chain of empty dirs with no .git anywhere; findGitRoot should
	// return "" and analyzeStaticDocs should not invent presence.
	parent := t.TempDir()
	sub := filepath.Join(parent, "core")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	// Sanity: parent of t.TempDir() can occasionally be inside a git repo on
	// the developer machine. Skip if so to keep the test hermetic.
	if findGitRoot(sub) != "" {
		t.Skip("ambient git repo above TempDir would mask the no-fallback case")
	}

	metrics := &types.C4Metrics{}
	analyzeStaticDocs(sub, metrics)

	if metrics.ReadmePresent || metrics.ChangelogPresent || metrics.ContributingPresent ||
		metrics.ExamplesPresent || metrics.DiagramsPresent {
		t.Errorf("expected all flags false outside a git repo, got %+v", metrics)
	}
}

func TestAnalyzeGoComments(t *testing.T) {
	dir := t.TempDir()

	// Create Go file with comments
	goContent := `package main

// This is a comment
// Another comment
func main() {
    /* Block comment
       spanning lines */
    fmt.Println("Hello")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goContent), 0644); err != nil {
		t.Fatal(err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  dir,
		Files: []types.SourceFile{
			{Path: filepath.Join(dir, "main.go"), Class: types.ClassSource},
		},
	}

	totalLines, commentLines := analyzeGoComments(target)
	if totalLines != 10 {
		t.Errorf("expected totalLines=10, got %d", totalLines)
	}
	// 2 single-line comments + 2 block comment lines = 4
	if commentLines != 4 {
		t.Errorf("expected commentLines=4, got %d", commentLines)
	}
}

func TestAnalyzeGoAPIDocs(t *testing.T) {
	dir := t.TempDir()

	// Create Go file with mix of documented and undocumented exports
	goContent := `package example

// PublicFunc is documented.
func PublicFunc() {}

func privateFunc() {}

// AnotherPublic has a doc.
func AnotherPublic() {}

func NoDoc() {}

// MyType is a documented type.
type MyType struct {}

type UndocType struct {}
`
	if err := os.WriteFile(filepath.Join(dir, "example.go"), []byte(goContent), 0644); err != nil {
		t.Fatal(err)
	}

	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  dir,
		Files: []types.SourceFile{
			{Path: filepath.Join(dir, "example.go"), Class: types.ClassSource},
		},
	}

	publicAPIs, documentedAPIs := analyzeGoAPIDocs(target)
	// PublicFunc, AnotherPublic, NoDoc, MyType, UndocType = 5 public
	if publicAPIs != 5 {
		t.Errorf("expected publicAPIs=5, got %d", publicAPIs)
	}
	// PublicFunc, AnotherPublic, MyType = 3 documented
	if documentedAPIs != 3 {
		t.Errorf("expected documentedAPIs=3, got %d", documentedAPIs)
	}
}

func TestC4Analyzer_Category(t *testing.T) {
	dir := t.TempDir()
	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  dir,
	}

	a := NewC4Analyzer(nil)
	result, err := a.Analyze([]*types.AnalysisTarget{target})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if result.Category != "C4" {
		t.Errorf("Category = %q, want %q", result.Category, "C4")
	}
	if result.Name != "C4: Documentation Quality" {
		t.Errorf("Name = %q, want %q", result.Name, "C4: Documentation Quality")
	}
}

func TestAnalyzeCommentDensity_Python(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	dir := t.TempDir()

	// Create Python file with comments
	pyContent := `# Header comment
# Another comment

def public_func():
    # inline comment
    pass

class MyClass:
    """Docstring for MyClass."""
    pass
`
	if err := os.WriteFile(filepath.Join(dir, "example.py"), []byte(pyContent), 0644); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "example.py"))
	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  dir,
		Files: []types.SourceFile{
			{
				Path:    filepath.Join(dir, "example.py"),
				Class:   types.ClassSource,
				Content: content,
			},
		},
	}

	totalLines, commentLines := analyzePythonComments(target, tsParser)
	if totalLines != 11 {
		t.Errorf("expected totalLines=11, got %d", totalLines)
	}
	// 3 # comments + possibly docstring lines (depends on Tree-sitter)
	if commentLines < 3 {
		t.Errorf("expected at least 3 commentLines, got %d", commentLines)
	}
}

func TestC4Analyzer_Integration(t *testing.T) {
	// Use real ARS repo for integration test
	root := findProjectRoot(t)

	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	a := NewC4Analyzer(tsParser)
	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  root,
	}

	// Populate files for Go analysis
	goFiles := []types.SourceFile{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".go" && !isVendorPath(path) && !isTestFile(path) {
			goFiles = append(goFiles, types.SourceFile{
				Path:  path,
				Class: types.ClassSource,
			})
		}
		return nil
	})
	target.Files = goFiles

	result, err := a.Analyze([]*types.AnalysisTarget{target})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	c4, ok := result.Metrics["c4"].(*types.C4Metrics)
	if !ok {
		t.Fatal("expected *types.C4Metrics in Metrics[\"c4\"]")
	}

	// ARS repo should have README.md
	if !c4.ReadmePresent {
		t.Error("expected ReadmePresent=true for ARS repo")
	}
	if c4.ReadmeWordCount == 0 {
		t.Error("expected ReadmeWordCount > 0 for ARS repo")
	}

	// Should have some Go files with comments
	if c4.TotalSourceLines == 0 {
		t.Error("expected TotalSourceLines > 0 for ARS repo")
	}
}

func TestAnalyzeCommentDensity_TypeScript(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	dir := t.TempDir()

	// Create TypeScript file with comments
	tsContent := `// Single line comment
// Another comment

/**
 * JSDoc comment
 */
export function myFunc(): void {
    // inline comment
    console.log("hello");
}

/* Block comment */
export class MyClass {
    private value: number;
}
`
	if err := os.WriteFile(filepath.Join(dir, "example.ts"), []byte(tsContent), 0644); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "example.ts"))
	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  dir,
		Files: []types.SourceFile{
			{
				Path:    filepath.Join(dir, "example.ts"),
				Class:   types.ClassSource,
				Content: content,
			},
		},
	}

	totalLines, commentLines := analyzeTypeScriptComments(target, tsParser)
	// Expect 16-17 lines depending on trailing newline handling
	if totalLines < 16 || totalLines > 17 {
		t.Errorf("expected totalLines=16-17, got %d", totalLines)
	}
	// At least 4 comment lines (single line comments + inline)
	if commentLines < 4 {
		t.Errorf("expected at least 4 commentLines, got %d", commentLines)
	}
}

func TestAnalyzeAPIDocs_Python(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	dir := t.TempDir()

	// Create Python file with mix of documented and undocumented functions
	pyContent := `def public_func():
    """This function is documented."""
    pass

def another_public():
    pass

class MyClass:
    """Documented class."""
    pass

class NoDocClass:
    pass

def _private_func():
    """This is private, should not be counted."""
    pass
`
	if err := os.WriteFile(filepath.Join(dir, "example.py"), []byte(pyContent), 0644); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "example.py"))
	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  dir,
		Files: []types.SourceFile{
			{
				Path:    filepath.Join(dir, "example.py"),
				Class:   types.ClassSource,
				Content: content,
			},
		},
	}

	publicAPIs, documentedAPIs := analyzePythonAPIDocs(target, tsParser)
	// public_func, another_public, MyClass, NoDocClass = 4 public (excluding _private_func)
	if publicAPIs != 4 {
		t.Errorf("expected publicAPIs=4, got %d", publicAPIs)
	}
	// public_func, MyClass = 2 documented
	if documentedAPIs != 2 {
		t.Errorf("expected documentedAPIs=2, got %d", documentedAPIs)
	}
}

func TestAnalyzeAPIDocs_TypeScript(t *testing.T) {
	dir := t.TempDir()

	// Create TypeScript file with mix of documented and undocumented exports
	tsContent := `/**
 * Documented function
 */
export function documentedFunc(): void {}

export function undocumentedFunc(): void {}

/**
 * Documented class
 */
export class MyClass {}

export interface UndocumentedInterface {}

/**
 * Documented type
 */
export type MyType = string;
`
	if err := os.WriteFile(filepath.Join(dir, "example.ts"), []byte(tsContent), 0644); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "example.ts"))
	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  dir,
		Files: []types.SourceFile{
			{
				Path:    filepath.Join(dir, "example.ts"),
				Class:   types.ClassSource,
				Content: content,
			},
		},
	}

	// We use a simpler regex-based approach for TypeScript
	publicAPIs, documentedAPIs := analyzeTypeScriptAPIDocs(target, nil)
	// 5 exports: documentedFunc, undocumentedFunc, MyClass, UndocumentedInterface, MyType
	if publicAPIs != 5 {
		t.Errorf("expected publicAPIs=5, got %d", publicAPIs)
	}
	// 3 documented: documentedFunc, MyClass, MyType
	if documentedAPIs != 3 {
		t.Errorf("expected documentedAPIs=3, got %d", documentedAPIs)
	}
}

// ---------------------------------------------------------------------------
// Tests for previously uncovered functions
// ---------------------------------------------------------------------------

func TestSetEvaluator(t *testing.T) {
	a := NewC4Analyzer(nil)
	if a.evaluator != nil {
		t.Fatal("expected evaluator to be nil initially")
	}
	a.SetEvaluator(nil) // nil is valid; just exercises the setter
	if a.evaluator != nil {
		t.Error("expected evaluator to remain nil after SetEvaluator(nil)")
	}
}

func TestAnalyzeTargetCodeMetrics_Go(t *testing.T) {
	dir := t.TempDir()
	goContent := `package example

// PublicFunc is documented.
func PublicFunc() {}

func privateFunc() {}
`
	if err := os.WriteFile(filepath.Join(dir, "example.go"), []byte(goContent), 0644); err != nil {
		t.Fatal(err)
	}

	a := NewC4Analyzer(nil)
	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  dir,
		Files: []types.SourceFile{
			{Path: filepath.Join(dir, "example.go"), Class: types.ClassSource},
		},
	}

	totalLines, commentLines, publicAPIs, documentedAPIs := a.analyzeTargetCodeMetrics(target)
	if totalLines == 0 {
		t.Error("expected totalLines > 0 for Go target")
	}
	if commentLines == 0 {
		t.Error("expected commentLines > 0 for Go target with doc comment")
	}
	if publicAPIs == 0 {
		t.Error("expected publicAPIs > 0 for Go target")
	}
	if documentedAPIs == 0 {
		t.Error("expected documentedAPIs > 0 for Go target")
	}
}

func TestAnalyzeTargetCodeMetrics_Python(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	dir := t.TempDir()
	pyContent := `# comment
def public_func():
    """Documented."""
    pass
`
	if err := os.WriteFile(filepath.Join(dir, "example.py"), []byte(pyContent), 0644); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "example.py"))

	a := NewC4Analyzer(tsParser)
	target := &types.AnalysisTarget{
		Language: types.LangPython,
		RootDir:  dir,
		Files: []types.SourceFile{
			{Path: filepath.Join(dir, "example.py"), Class: types.ClassSource, Content: content},
		},
	}

	totalLines, commentLines, publicAPIs, documentedAPIs := a.analyzeTargetCodeMetrics(target)
	if totalLines == 0 {
		t.Error("expected totalLines > 0 for Python target")
	}
	_ = commentLines
	if publicAPIs == 0 {
		t.Error("expected publicAPIs > 0 for Python target")
	}
	if documentedAPIs == 0 {
		t.Error("expected documentedAPIs > 0 for Python target")
	}
}

func TestAnalyzeTargetCodeMetrics_TypeScript(t *testing.T) {
	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	dir := t.TempDir()
	tsContent := `// comment
/** Doc */
export function myFunc(): void {}
`
	if err := os.WriteFile(filepath.Join(dir, "example.ts"), []byte(tsContent), 0644); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(dir, "example.ts"))

	a := NewC4Analyzer(tsParser)
	target := &types.AnalysisTarget{
		Language: types.LangTypeScript,
		RootDir:  dir,
		Files: []types.SourceFile{
			{Path: filepath.Join(dir, "example.ts"), Class: types.ClassSource, Content: content},
		},
	}

	totalLines, commentLines, publicAPIs, documentedAPIs := a.analyzeTargetCodeMetrics(target)
	if totalLines == 0 {
		t.Error("expected totalLines > 0 for TypeScript target")
	}
	_ = commentLines
	if publicAPIs == 0 {
		t.Error("expected publicAPIs > 0 for TypeScript target")
	}
	_ = documentedAPIs
}

func TestAnalyzeTargetCodeMetrics_NilParserSkipsPythonTS(t *testing.T) {
	a := NewC4Analyzer(nil) // no tree-sitter parser

	dir := t.TempDir()
	pyTarget := &types.AnalysisTarget{Language: types.LangPython, RootDir: dir}
	tl, cl, pa, da := a.analyzeTargetCodeMetrics(pyTarget)
	if tl != 0 || cl != 0 || pa != 0 || da != 0 {
		t.Errorf("expected all zeros for Python with nil parser, got %d %d %d %d", tl, cl, pa, da)
	}

	tsTarget := &types.AnalysisTarget{Language: types.LangTypeScript, RootDir: dir}
	tl, cl, pa, da = a.analyzeTargetCodeMetrics(tsTarget)
	if tl != 0 || cl != 0 || pa != 0 || da != 0 {
		t.Errorf("expected all zeros for TypeScript with nil parser, got %d %d %d %d", tl, cl, pa, da)
	}
}

func TestReadReadmeContent(t *testing.T) {
	dir := t.TempDir()

	// No README
	got := readReadmeContent(dir)
	if got != "" {
		t.Errorf("expected empty string when no README, got %q", got)
	}

	// README.md present
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	got = readReadmeContent(dir)
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestReadReadmeContent_Truncation(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("a", readmeTruncateLimit+100)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(big), 0644); err != nil {
		t.Fatal(err)
	}
	got := readReadmeContent(dir)
	if len(got) != readmeTruncateLimit {
		t.Errorf("expected content truncated to %d, got %d", readmeTruncateLimit, len(got))
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	content := "before\n```go\nfunc main() {}\n```\nafter\n```\nplain block\n```\n"
	blocks := extractCodeBlocks(content)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0], "func main()") {
		t.Errorf("first block should contain 'func main()', got %q", blocks[0])
	}
	if !strings.Contains(blocks[1], "plain block") {
		t.Errorf("second block should contain 'plain block', got %q", blocks[1])
	}
}

func TestExtractCodeBlocks_Empty(t *testing.T) {
	blocks := extractCodeBlocks("")
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for empty content, got %d", len(blocks))
	}
}

func TestTruncateExampleContent(t *testing.T) {
	short := "hello"
	if truncateExampleContent(short) != short {
		t.Error("short content should not be truncated")
	}

	big := strings.Repeat("x", exampleContentLimit+50)
	truncated := truncateExampleContent(big)
	if len(truncated) != exampleContentLimit {
		t.Errorf("expected truncation to %d, got %d", exampleContentLimit, len(truncated))
	}
}

func TestEstimateTokens(t *testing.T) {
	content := strings.Repeat("a", 400)
	tokens := estimateTokens(content)
	if tokens != 100 {
		t.Errorf("expected 100 tokens for 400 chars, got %d", tokens)
	}
	if estimateTokens("") != 0 {
		t.Error("expected 0 tokens for empty string")
	}
}

func TestCollectDocsSummary(t *testing.T) {
	dir := t.TempDir()
	metrics := &types.C4Metrics{
		ReadmePresent:       true,
		ReadmeWordCount:     150,
		ChangelogPresent:    true,
		ContributingPresent: false,
		ExamplesPresent:     true,
		DiagramsPresent:     false,
		CommentDensity:      12.5,
		APIDocCoverage:      80.0,
		PublicAPIs:          10,
		DocumentedAPIs:      8,
	}

	summary := collectDocsSummary(dir, metrics)
	if !strings.Contains(summary, "README: Present") {
		t.Error("expected summary to mention README present")
	}
	if !strings.Contains(summary, "CHANGELOG: Present") {
		t.Error("expected summary to mention CHANGELOG present")
	}
	if !strings.Contains(summary, "CONTRIBUTING: NOT PRESENT") {
		t.Error("expected summary to mention CONTRIBUTING missing")
	}
	if !strings.Contains(summary, "12.5") {
		t.Error("expected summary to include comment density")
	}
}

func TestCollectDocsSummary_WithReadme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test README\nSome content here."), 0644); err != nil {
		t.Fatal(err)
	}
	metrics := &types.C4Metrics{ReadmePresent: true, ReadmeWordCount: 5}
	summary := collectDocsSummary(dir, metrics)
	if !strings.Contains(summary, "README Content") {
		t.Error("expected summary to include README content section")
	}
}

func TestCountSampledFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	count := countSampledFiles(dir)
	if count != 0 {
		t.Errorf("expected 0 sampled files in empty dir, got %d", count)
	}
}

func TestCountSampledFiles_WithReadme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	count := countSampledFiles(dir)
	if count != 1 {
		t.Errorf("expected 1 sampled file (README), got %d", count)
	}
}

func TestCountSampledFiles_WithExamples(t *testing.T) {
	dir := t.TempDir()
	// README
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	// examples dir with files
	exDir := filepath.Join(dir, "examples")
	if err := os.Mkdir(exDir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		name := filepath.Join(exDir, strings.Repeat("f", i+1)+".go")
		if err := os.WriteFile(name, []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	count := countSampledFiles(dir)
	if count != 4 { // 1 README + 3 example files
		t.Errorf("expected 4 sampled files, got %d", count)
	}
}

func TestCollectExampleContent_NoExamples(t *testing.T) {
	dir := t.TempDir()
	content := collectExampleContent(dir)
	if content != "" {
		t.Errorf("expected empty content when no examples, got %q", content)
	}
}

func TestCollectExampleContent_FromDirectory(t *testing.T) {
	dir := t.TempDir()
	exDir := filepath.Join(dir, "examples")
	if err := os.Mkdir(exDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exDir, "demo.go"), []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	content := collectExampleContent(dir)
	if !strings.Contains(content, "demo.go") {
		t.Error("expected content to reference demo.go example file")
	}
}

func TestCollectExampleContent_FromReadme(t *testing.T) {
	dir := t.TempDir()
	readme := "# Title\n\n```go\nfunc hello() {}\n```\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatal(err)
	}

	content := collectExampleContent(dir)
	if !strings.Contains(content, "README Example") {
		t.Error("expected content to include README examples")
	}
}

func TestC4Analyzer_RealRepo_MetricRanges(t *testing.T) {
	root := findProjectRoot(t)

	tsParser, err := parser.NewTreeSitterParser()
	if err != nil {
		t.Skip("Tree-sitter not available:", err)
	}
	defer tsParser.Close()

	a := NewC4Analyzer(tsParser)
	target := &types.AnalysisTarget{
		Language: types.LangGo,
		RootDir:  root,
	}

	// Populate files for Go analysis
	goFiles := []types.SourceFile{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".go" && !isVendorPath(path) && !isTestFile(path) {
			goFiles = append(goFiles, types.SourceFile{
				Path:  path,
				Class: types.ClassSource,
			})
		}
		return nil
	})
	target.Files = goFiles

	result, err := a.Analyze([]*types.AnalysisTarget{target})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	c4 := result.Metrics["c4"].(*types.C4Metrics)

	// CommentDensity: 0-100 percentage
	if c4.CommentDensity < 0 || c4.CommentDensity > 100 {
		t.Errorf("CommentDensity = %f, expected 0-100", c4.CommentDensity)
	}

	// APIDocCoverage: 0-100 percentage
	if c4.APIDocCoverage < 0 || c4.APIDocCoverage > 100 {
		t.Errorf("APIDocCoverage = %f, expected 0-100", c4.APIDocCoverage)
	}

	// ReadmeWordCount: >= 0
	if c4.ReadmeWordCount < 0 {
		t.Errorf("ReadmeWordCount = %d, expected >= 0", c4.ReadmeWordCount)
	}

	// TotalSourceLines: >= 0
	if c4.TotalSourceLines < 0 {
		t.Errorf("TotalSourceLines = %d, expected >= 0", c4.TotalSourceLines)
	}

	// PublicAPIs: >= 0
	if c4.PublicAPIs < 0 {
		t.Errorf("PublicAPIs = %d, expected >= 0", c4.PublicAPIs)
	}
}
