// Package c3 analyzes C3 (Architecture) metrics for agent-readiness.
//
// C3 measures architectural complexity: directory depth, module fanout, circular
// dependencies, and dead code. Poor architecture forces agents to understand
// large dependency graphs before making changes, increasing error rates.
//
// TypeScript-specific challenges:
// - Imports use relative paths ("./" "../") unlike Go's package-based imports
// - tsconfig.json path aliases ("@/components") require resolution
// - node_modules imports must be filtered (external dependencies)
// - Index files (index.ts) create multiple valid import paths for same module
// - ESM (import/export) and CommonJS (require/module.exports) both common
//
// This file uses Tree-sitter parsing since TypeScript lacks a standard AST API
// like Go's go/packages or Python's ast module.
package c3

import (
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// tsFilterSourceFiles filters to source-only TypeScript files (not test files).
func tsFilterSourceFiles(files []*parser.ParsedTreeSitterFile) []*parser.ParsedTreeSitterFile {
	var result []*parser.ParsedTreeSitterFile
	for _, f := range files {
		if tsIsTestFile(f.RelPath) {
			continue
		}
		result = append(result, f)
	}
	return result
}

// tsIsTestFile checks if a TypeScript file path indicates a test file.
func tsIsTestFile(path string) bool {
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

// tsBuildImportGraph builds an import graph from TypeScript files.
//
// It tracks intra-project imports only (skips node_modules/third-party).
//
// Circular dependency detection uses Tarjan's strongly connected components algorithm
// (implemented in shared.DetectCycles). Tarjan's runs in O(V+E) time - linear in
// graph size - making it efficient even for large codebases (1000+ files).
//
// Why circular dependencies matter for agents: Agents struggle to reason about
// circular deps because they create bidirectional knowledge requirements. To modify
// module A, the agent must understand module B, which requires understanding A,
// creating an infinite reasoning loop. Breaking cycles into DAGs (directed acyclic
// graphs) allows agents to reason bottom-up with confidence.
func tsBuildImportGraph(files []*parser.ParsedTreeSitterFile) *shared.ImportGraph {
	g := &shared.ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	knownFiles := tsBuildKnownFilesMap(files)

	for _, f := range files {
		root := f.Tree.RootNode()
		fromFile := tsNormalizePath(f.RelPath)
		fromDir := filepath.Dir(f.RelPath)
		tsExtractImports(root, f.Content, fromFile, fromDir, knownFiles, g)
	}

	return g
}

func tsBuildKnownFilesMap(files []*parser.ParsedTreeSitterFile) map[string]string {
	knownFiles := make(map[string]string)
	for _, f := range files {
		normalized := tsNormalizePath(f.RelPath)
		knownFiles[normalized] = f.RelPath
	}
	return knownFiles
}

func tsExtractImports(root *tree_sitter.Node, content []byte, fromFile string, fromDir string, knownFiles map[string]string, g *shared.ImportGraph) {
	shared.WalkTree(root, func(node *tree_sitter.Node) {
		modulePath := tsExtractModulePath(node, content)
		if modulePath == "" || !strings.HasPrefix(modulePath, ".") {
			return
		}
		tsAddImportEdge(modulePath, fromFile, fromDir, knownFiles, g)
	})
}

func tsExtractModulePath(node *tree_sitter.Node, content []byte) string {
	kind := node.Kind()
	switch kind {
	case "import_statement":
		src := node.ChildByFieldName("source")
		if src != nil {
			return tsStripQuotes(shared.NodeText(src, content))
		}
	case "call_expression":
		fn := node.ChildByFieldName("function")
		if fn == nil || shared.NodeText(fn, content) != "require" {
			return ""
		}
		args := node.ChildByFieldName("arguments")
		if args == nil {
			return ""
		}
		for i := uint(0); i < args.ChildCount(); i++ {
			child := args.Child(i)
			if child != nil && child.Kind() == "string" {
				return tsStripQuotes(shared.NodeText(child, content))
			}
		}
	}
	return ""
}

func tsAddImportEdge(modulePath string, fromFile string, fromDir string, knownFiles map[string]string, g *shared.ImportGraph) {
	resolved := filepath.Join(fromDir, modulePath)
	resolved = filepath.Clean(resolved)
	resolved = strings.ReplaceAll(resolved, string(os.PathSeparator), "/")
	normalizedResolved := tsNormalizePath(resolved)
	if _, ok := knownFiles[normalizedResolved]; ok {
		if normalizedResolved != fromFile {
			g.Forward[fromFile] = appendUnique(g.Forward[fromFile], normalizedResolved)
			g.Reverse[normalizedResolved] = appendUnique(g.Reverse[normalizedResolved], fromFile)
		}
	}
}

// tsNormalizePath normalizes a TypeScript file path for import graph matching.
// Strips .ts/.tsx/.js extensions and normalizes separators.
func tsNormalizePath(p string) string {
	p = strings.ReplaceAll(p, string(os.PathSeparator), "/")
	for _, ext := range []string{".ts", ".tsx", ".js", ".jsx"} {
		p = strings.TrimSuffix(p, ext)
	}
	// Handle index files
	p = strings.TrimSuffix(p, "/index")
	return p
}

// tsStripQuotes removes surrounding quotes from a string literal.
func tsStripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// tsDetectDeadCode finds exported symbols not imported by other project files.
//
// Why track dead exports for agent-readiness: Dead exports bloat the API surface
// agents must understand. When an agent sees 50 exported functions but only 10
// are actually used, it wastes reasoning capacity on irrelevant code. Pruning
// dead exports reduces cognitive load and improves agent focus on active APIs.
//
// Limitation: Test files are excluded from the "imported by" check because test
// imports don't represent production usage. This may flag some exports as dead
// when they're only used in tests - acceptable trade-off for simpler analysis.
func tsDetectDeadCode(files []*parser.ParsedTreeSitterFile) []types.DeadExport {
	if len(files) <= 1 {
		return nil // Single file: no cross-file analysis possible
	}

	var defs []tsExportDef

	for _, f := range files {
		if tsIsTestFile(f.RelPath) {
			continue
		}
		root := f.Tree.RootNode()
		for i := uint(0); i < root.ChildCount(); i++ {
			child := root.Child(i)
			if child == nil {
				continue
			}

			kind := child.Kind()

			// Handle export_statement
			if kind == "export_statement" {
				tsCollectExportedDefs(child, f.Content, f.RelPath, &defs)
				continue
			}
		}
	}

	// Collect all imported names across files
	importedNames := make(map[string]bool)
	for _, f := range files {
		root := f.Tree.RootNode()
		shared.WalkTree(root, func(node *tree_sitter.Node) {
			if node.Kind() != "import_statement" {
				return
			}
			// Collect imported identifiers from import clause
			tsCollectImportedNames(node, f.Content, importedNames)
		})
	}

	// Flag definitions not imported by any other file
	var dead []types.DeadExport
	for _, d := range defs {
		if !importedNames[d.name] {
			dead = append(dead, types.DeadExport{
				Package: "",
				Name:    d.name,
				File:    filepath.Base(d.file),
				Line:    d.line,
				Kind:    d.kind,
			})
		}
	}

	return dead
}

// tsExportDef represents an exported definition found during dead code detection.
type tsExportDef struct {
	name string
	file string
	line int
	kind string
}

// tsCollectExportedDefs collects exported function/class/variable definitions from an export statement.
func tsCollectExportedDefs(exportNode *tree_sitter.Node, content []byte, relPath string, defs *[]tsExportDef) {
	for i := uint(0); i < exportNode.ChildCount(); i++ {
		child := exportNode.Child(i)
		if child == nil {
			continue
		}

		childKind := child.Kind()
		switch childKind {
		case "function_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				*defs = append(*defs, tsExportDef{
					name: shared.NodeText(nameNode, content),
					file: relPath,
					line: int(nameNode.StartPosition().Row) + 1,
					kind: "func",
				})
			}
		case "class_declaration":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				*defs = append(*defs, tsExportDef{
					name: shared.NodeText(nameNode, content),
					file: relPath,
					line: int(nameNode.StartPosition().Row) + 1,
					kind: "type",
				})
			}
		case "lexical_declaration":
			// export const foo = ...
			for j := uint(0); j < child.ChildCount(); j++ {
				declChild := child.Child(j)
				if declChild != nil && declChild.Kind() == "variable_declarator" {
					nameNode := declChild.ChildByFieldName("name")
					if nameNode != nil {
						*defs = append(*defs, tsExportDef{
							name: shared.NodeText(nameNode, content),
							file: relPath,
							line: int(nameNode.StartPosition().Row) + 1,
							kind: "var",
						})
					}
				}
			}
		case "export_clause":
			// export { foo, bar }
			for j := uint(0); j < child.ChildCount(); j++ {
				spec := child.Child(j)
				if spec != nil && spec.Kind() == "export_specifier" {
					nameNode := spec.ChildByFieldName("name")
					if nameNode != nil {
						*defs = append(*defs, tsExportDef{
							name: shared.NodeText(nameNode, content),
							file: relPath,
							line: int(nameNode.StartPosition().Row) + 1,
							kind: "var",
						})
					}
				}
			}
		}
	}
}

// tsCollectImportedNames collects imported identifiers from an import statement.
func tsCollectImportedNames(importNode *tree_sitter.Node, content []byte, names map[string]bool) {
	for i := uint(0); i < importNode.ChildCount(); i++ {
		child := importNode.Child(i)
		if child == nil {
			continue
		}

		childKind := child.Kind()
		switch childKind {
		case "import_clause":
			for j := uint(0); j < child.ChildCount(); j++ {
				inner := child.Child(j)
				if inner == nil {
					continue
				}
				switch inner.Kind() {
				case "identifier":
					names[shared.NodeText(inner, content)] = true
				case "named_imports":
					for k := uint(0); k < inner.ChildCount(); k++ {
						spec := inner.Child(k)
						if spec != nil && spec.Kind() == "import_specifier" {
							nameNode := spec.ChildByFieldName("name")
							if nameNode != nil {
								names[shared.NodeText(nameNode, content)] = true
							}
						}
					}
				case "namespace_import":
					// import * as foo
					nameNode := inner.ChildByFieldName("name")
					if nameNode == nil {
						// fallback: look for identifier child
						for k := uint(0); k < inner.ChildCount(); k++ {
							c := inner.Child(k)
							if c != nil && c.Kind() == "identifier" {
								names[shared.NodeText(c, content)] = true
							}
						}
					} else {
						names[shared.NodeText(nameNode, content)] = true
					}
				}
			}
		}
	}
}

// tsAnalyzeDirectoryDepth computes max and average directory depth from TypeScript file paths.
func tsAnalyzeDirectoryDepth(files []*parser.ParsedTreeSitterFile, rootDir string) (int, float64) {
	if len(files) == 0 {
		return 0, 0
	}

	maxDepth := 0
	totalDepth := 0

	for _, f := range files {
		relPath := f.RelPath
		if relPath == "" && rootDir != "" {
			var err error
			relPath, err = filepath.Rel(rootDir, f.Path)
			if err != nil {
				continue
			}
		}

		// Count directory separators
		depth := strings.Count(relPath, "/")
		if os.PathSeparator != '/' {
			depth += strings.Count(relPath, string(os.PathSeparator))
		}

		totalDepth += depth
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	avg := float64(totalDepth) / float64(len(files))
	return maxDepth, avg
}
