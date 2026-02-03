// Package analyzer provides code analysis implementations for the ARS pipeline.
package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	tsp "github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C4Analyzer implements the pipeline.Analyzer interface for C4: Documentation Quality.
// It analyzes README presence, comment density, API doc coverage, and other documentation artifacts.
type C4Analyzer struct {
	tsParser *tsp.TreeSitterParser
}

// NewC4Analyzer creates a C4Analyzer. Tree-sitter parser is needed for Python/TS analysis.
func NewC4Analyzer(tsParser *tsp.TreeSitterParser) *C4Analyzer {
	return &C4Analyzer{tsParser: tsParser}
}

// Name returns the analyzer display name.
func (a *C4Analyzer) Name() string {
	return "C4: Documentation Quality"
}

// Analyze runs the C4 documentation quality analysis.
func (a *C4Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}
	rootDir := targets[0].RootDir

	metrics := &types.C4Metrics{
		ChangelogDaysOld: -1, // Default to -1 (not present)
	}

	// C4-01: README presence and word count
	metrics.ReadmePresent, metrics.ReadmeWordCount = analyzeReadme(rootDir)

	// C4-04: CHANGELOG presence
	metrics.ChangelogPresent = analyzeChangelog(rootDir)

	// C4-05: Examples presence
	metrics.ExamplesPresent = analyzeExamples(rootDir)

	// C4-06: CONTRIBUTING presence
	metrics.ContributingPresent = analyzeContributing(rootDir)

	// C4-07: Diagrams presence
	metrics.DiagramsPresent = analyzeDiagrams(rootDir)

	// C4-02 & C4-03: Comment density and API doc coverage across all languages
	totalLines, commentLines := 0, 0
	publicAPIs, documentedAPIs := 0, 0

	for _, target := range targets {
		switch target.Language {
		case types.LangGo:
			tl, cl := analyzeGoComments(target)
			totalLines += tl
			commentLines += cl
			pa, da := analyzeGoAPIDocs(target)
			publicAPIs += pa
			documentedAPIs += da
		case types.LangPython:
			if a.tsParser != nil {
				tl, cl := analyzePythonComments(target, a.tsParser)
				totalLines += tl
				commentLines += cl
				pa, da := analyzePythonAPIDocs(target, a.tsParser)
				publicAPIs += pa
				documentedAPIs += da
			}
		case types.LangTypeScript:
			if a.tsParser != nil {
				tl, cl := analyzeTypeScriptComments(target, a.tsParser)
				totalLines += tl
				commentLines += cl
				pa, da := analyzeTypeScriptAPIDocs(target, a.tsParser)
				publicAPIs += pa
				documentedAPIs += da
			}
		}
	}

	metrics.TotalSourceLines = totalLines
	metrics.CommentLines = commentLines
	metrics.PublicAPIs = publicAPIs
	metrics.DocumentedAPIs = documentedAPIs

	if totalLines > 0 {
		metrics.CommentDensity = float64(commentLines) / float64(totalLines) * 100
	}
	if publicAPIs > 0 {
		metrics.APIDocCoverage = float64(documentedAPIs) / float64(publicAPIs) * 100
	}

	return &types.AnalysisResult{
		Name:     "C4: Documentation Quality",
		Category: "C4",
		Metrics:  map[string]interface{}{"c4": metrics},
	}, nil
}

// analyzeReadme checks for README presence and counts words.
func analyzeReadme(rootDir string) (bool, int) {
	readmePaths := []string{
		filepath.Join(rootDir, "README.md"),
		filepath.Join(rootDir, "README"),
		filepath.Join(rootDir, "readme.md"),
		filepath.Join(rootDir, "Readme.md"),
		filepath.Join(rootDir, "README.txt"),
	}

	for _, path := range readmePaths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		wordCount := countWords(string(content))
		return true, wordCount
	}

	return false, 0
}

// countWords counts words in text using unicode space detection.
func countWords(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}
	return words
}

// analyzeChangelog checks for CHANGELOG presence.
func analyzeChangelog(rootDir string) bool {
	changelogPaths := []string{
		filepath.Join(rootDir, "CHANGELOG.md"),
		filepath.Join(rootDir, "CHANGELOG"),
		filepath.Join(rootDir, "changelog.md"),
		filepath.Join(rootDir, "Changelog.md"),
		filepath.Join(rootDir, "HISTORY.md"),
		filepath.Join(rootDir, "CHANGES.md"),
	}

	for _, path := range changelogPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// analyzeExamples checks for examples directory or code blocks in README.
func analyzeExamples(rootDir string) bool {
	// Check for examples/ directory
	examplesDirs := []string{
		filepath.Join(rootDir, "examples"),
		filepath.Join(rootDir, "example"),
		filepath.Join(rootDir, "_examples"),
	}
	for _, dir := range examplesDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return true
		}
	}

	// Check for code blocks in README.md
	readmePath := filepath.Join(rootDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return false
	}

	// Count fenced code blocks (```)
	codeBlockPattern := regexp.MustCompile("(?m)^```")
	matches := codeBlockPattern.FindAllIndex(content, -1)
	// Need at least 2 matches (open and close) for a code block
	return len(matches) >= 2
}

// analyzeContributing checks for CONTRIBUTING presence.
func analyzeContributing(rootDir string) bool {
	contributingPaths := []string{
		filepath.Join(rootDir, "CONTRIBUTING.md"),
		filepath.Join(rootDir, "CONTRIBUTING"),
		filepath.Join(rootDir, "contributing.md"),
		filepath.Join(rootDir, ".github", "CONTRIBUTING.md"),
	}

	for _, path := range contributingPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// analyzeDiagrams checks for architecture diagrams in docs/.
func analyzeDiagrams(rootDir string) bool {
	docsDir := filepath.Join(rootDir, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		docsDir = rootDir // Fall back to root if no docs/ dir
	}

	diagramExtensions := []string{".png", ".svg", ".mermaid", ".drawio", ".puml"}
	diagramKeywords := []string{"architecture", "diagram", "flow", "sequence", "class", "er", "uml"}

	found := false
	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		name := strings.ToLower(filepath.Base(path))

		// Check for diagram file extensions
		for _, diagExt := range diagramExtensions {
			if ext == diagExt {
				// Check if filename contains diagram keywords
				for _, keyword := range diagramKeywords {
					if strings.Contains(name, keyword) {
						found = true
						return nil
					}
				}
			}
		}

		// Also check for mermaid blocks in markdown
		if ext == ".md" {
			content, err := os.ReadFile(path)
			if err == nil && bytes.Contains(content, []byte("```mermaid")) {
				found = true
			}
		}

		return nil
	})

	return found
}

// analyzeGoComments counts total lines and comment lines in Go files.
func analyzeGoComments(target *types.AnalysisTarget) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content, err := os.ReadFile(sf.Path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		inBlockComment := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				commentLines++
			} else if strings.HasPrefix(trimmed, "/*") {
				inBlockComment = true
				commentLines++
			} else if inBlockComment {
				commentLines++
				if strings.Contains(trimmed, "*/") {
					inBlockComment = false
				}
			}
		}
	}
	return
}

// analyzeGoAPIDocs counts exported (public) APIs and those with godoc comments.
func analyzeGoAPIDocs(target *types.AnalysisTarget) (publicAPIs, documentedAPIs int) {
	fset := token.NewFileSet()

	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		f, err := goparser.ParseFile(fset, sf.Path, nil, goparser.ParseComments)
		if err != nil {
			continue
		}

		// Build comment map for the file
		cmap := ast.NewCommentMap(fset, f, f.Comments)

		// Check exported functions, types, and methods
		ast.Inspect(f, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				if decl.Name.IsExported() {
					publicAPIs++
					// Check if there's a doc comment
					if decl.Doc != nil && len(decl.Doc.List) > 0 {
						documentedAPIs++
					} else if comments := cmap[decl]; len(comments) > 0 {
						documentedAPIs++
					}
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							publicAPIs++
							// Check for doc comment on the type or containing GenDecl
							if decl.Doc != nil && len(decl.Doc.List) > 0 {
								documentedAPIs++
							} else if s.Doc != nil && len(s.Doc.List) > 0 {
								documentedAPIs++
							}
						}
					}
				}
			}
			return true
		})
	}
	return
}

// analyzePythonComments counts total lines and comment lines in Python files.
func analyzePythonComments(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		// Parse with Tree-sitter to find comments
		tree, err := tsParser.ParseFile(types.LangPython, ".py", content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		commentLines += countTreeSitterCommentLines(root)
		tree.Close()
	}
	return
}

// countTreeSitterCommentLines counts comment lines in a Tree-sitter tree.
func countTreeSitterCommentLines(node *tree_sitter.Node) int {
	count := 0
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if n.Kind() == "comment" {
			// Count lines in the comment
			start := n.StartPosition()
			end := n.EndPosition()
			count += int(end.Row-start.Row) + 1
		}
		for i := uint(0); i < uint(n.ChildCount()); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}
	walk(node)
	return count
}

// analyzePythonAPIDocs counts public functions/classes and those with docstrings.
func analyzePythonAPIDocs(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (publicAPIs, documentedAPIs int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		tree, err := tsParser.ParseFile(types.LangPython, ".py", content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		pa, da := countPythonAPIDocs(root, content)
		publicAPIs += pa
		documentedAPIs += da
		tree.Close()
	}
	return
}

// countPythonAPIDocs walks the tree to find public functions/classes with docstrings.
func countPythonAPIDocs(root *tree_sitter.Node, content []byte) (publicAPIs, documentedAPIs int) {
	// Walk through direct children of module looking for function_definition and class_definition
	for i := uint(0); i < uint(root.ChildCount()); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "function_definition", "class_definition":
			name := extractPythonDefName(child, content)
			if !strings.HasPrefix(name, "_") { // Public if not starting with _
				publicAPIs++
				if hasPythonDocstring(child) {
					documentedAPIs++
				}
			}
		}
	}
	return
}

// extractPythonDefName extracts the name of a function or class definition.
func extractPythonDefName(node *tree_sitter.Node, content []byte) string {
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "identifier" {
			return child.Utf8Text(content)
		}
	}
	return ""
}

// hasPythonDocstring checks if a function/class has a docstring as its first statement.
func hasPythonDocstring(node *tree_sitter.Node) bool {
	// Look for block child (the body)
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "block" {
			// First non-trivial child of block might be expression_statement with string
			for j := uint(0); j < uint(child.ChildCount()); j++ {
				stmt := child.Child(j)
				if stmt == nil {
					continue
				}
				if stmt.Kind() == "expression_statement" {
					// Check if it contains a string
					for k := uint(0); k < uint(stmt.ChildCount()); k++ {
						expr := stmt.Child(k)
						if expr != nil && expr.Kind() == "string" {
							return true
						}
					}
				}
				// Only check first meaningful statement
				if stmt.Kind() != "comment" {
					return false
				}
			}
			return false
		}
	}
	return false
}

// analyzeTypeScriptComments counts total lines and comment lines in TypeScript files.
func analyzeTypeScriptComments(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := tsParser.ParseFile(types.LangTypeScript, ext, content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		commentLines += countTreeSitterCommentLines(root)
		tree.Close()
	}
	return
}

// analyzeTypeScriptAPIDocs counts exported functions/classes and those with JSDoc.
func analyzeTypeScriptAPIDocs(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (publicAPIs, documentedAPIs int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		// Use simpler regex-based approach for export detection
		lines := bytes.Split(content, []byte("\n"))
		inJSDoc := false
		hasJSDoc := false

		for _, line := range lines {
			lineStr := string(bytes.TrimSpace(line))

			// Track JSDoc comments
			if strings.HasPrefix(lineStr, "/**") {
				inJSDoc = true
				hasJSDoc = true
			}
			if inJSDoc && strings.Contains(lineStr, "*/") {
				inJSDoc = false
			}

			// Check for export declarations
			if strings.HasPrefix(lineStr, "export ") {
				if strings.Contains(lineStr, "function ") ||
					strings.Contains(lineStr, "class ") ||
					strings.Contains(lineStr, "const ") ||
					strings.Contains(lineStr, "interface ") ||
					strings.Contains(lineStr, "type ") {
					publicAPIs++
					if hasJSDoc {
						documentedAPIs++
					}
				}
				hasJSDoc = false
			} else if !strings.HasPrefix(lineStr, "*") && !strings.HasPrefix(lineStr, "//") && lineStr != "" {
				// Non-comment, non-export line resets JSDoc tracking
				if !inJSDoc {
					hasJSDoc = false
				}
			}
		}
	}
	return
}
