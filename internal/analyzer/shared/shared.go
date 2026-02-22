// Package shared provides common utilities for analyzer implementations.
// This package is separate from the main analyzer package to avoid import cycles
// when subdirectory packages need these utilities.
package shared

import (
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
)

// ImportGraph holds forward and reverse adjacency lists for intra-module imports.
type ImportGraph struct {
	Forward map[string][]string // package -> packages it imports (efferent)
	Reverse map[string][]string // package -> packages that import it (afferent)
}

// BuildImportGraph constructs an import graph from parsed packages, filtering
// to only intra-module imports (those with the given module path prefix).
func BuildImportGraph(pkgs []*parser.ParsedPackage, modulePath string) *ImportGraph {
	g := &ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			continue // skip test packages for coupling analysis
		}
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, modulePath) {
				g.Forward[pkg.PkgPath] = append(g.Forward[pkg.PkgPath], importPath)
				g.Reverse[importPath] = append(g.Reverse[importPath], pkg.PkgPath)
			}
		}
	}

	return g
}

// WalkTree walks a Tree-sitter tree depth-first, calling fn for each node.
func WalkTree(node *tree_sitter.Node, fn func(*tree_sitter.Node)) {
	if node == nil {
		return
	}
	fn(node)
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			WalkTree(child, fn)
		}
	}
}

// NodeText extracts the text content of a Tree-sitter node.
func NodeText(node *tree_sitter.Node, content []byte) string {
	return string(content[node.StartByte():node.EndByte()])
}

// CountLines counts lines in source content.
func CountLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := 1
	for _, b := range content {
		if b == '\n' {
			count++
		}
	}
	return count
}

// IsTestFileByPath checks if a file path indicates a Python test file.
// Used by C6 testing analyzer.
func IsTestFileByPath(path string) bool {
	base := strings.ToLower(path)
	parts := strings.Split(base, "/")
	if len(parts) > 0 {
		base = parts[len(parts)-1]
	}

	return strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(base, "_test.py") ||
		base == "conftest.py"
}

// TsIsTestFile checks if a TypeScript file path indicates a test file.
// Used by C6 testing analyzer.
func TsIsTestFile(path string) bool {
	lower := strings.ToLower(path)
	base := lower
	parts := strings.Split(lower, "/")
	if len(parts) > 0 {
		base = parts[len(parts)-1]
	}

	// Check __tests__ directory
	for _, p := range parts {
		if p == "__tests__" {
			return true
		}
	}

	return strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.js")
}

// TsStripQuotes removes surrounding quotes from a string literal.
// Used by C6 testing analyzer.
func TsStripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// FilterTreeSitterFiles returns files for which isTest(relPath) is false.
// Used by C1 and C3 analyzers to strip test files before source analysis.
func FilterTreeSitterFiles(files []*parser.ParsedTreeSitterFile, isTest func(relPath string) bool) []*parser.ParsedTreeSitterFile {
	var result []*parser.ParsedTreeSitterFile
	for _, f := range files {
		if !isTest(f.RelPath) {
			result = append(result, f)
		}
	}
	return result
}

// AnalyzeDirectoryDepth computes max and average directory depth from tree-sitter file paths.
// Used by C3 architecture analyzers for Python and TypeScript.
func AnalyzeDirectoryDepth(files []*parser.ParsedTreeSitterFile, rootDir string) (maxDepth int, avgDepth float64) {
	if len(files) == 0 {
		return 0, 0
	}

	totalDepth := 0
	for _, f := range files {
		relPath := f.RelPath
		if relPath == "" && rootDir != "" {
			if rel, err := filepath.Rel(rootDir, f.Path); err == nil {
				relPath = rel
			} else {
				continue
			}
		}
		depth := strings.Count(relPath, "/")
		if os.PathSeparator != '/' {
			depth += strings.Count(relPath, string(os.PathSeparator))
		}
		totalDepth += depth
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth, float64(totalDepth) / float64(len(files))
}
