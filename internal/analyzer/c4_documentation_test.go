package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

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

func isVendorPath(path string) bool {
	return filepath.Base(filepath.Dir(path)) == "vendor"
}

func isTestFile(path string) bool {
	return filepath.Base(path) != "c4_documentation_test.go" &&
		len(path) > 8 && path[len(path)-8:] == "_test.go"
}
