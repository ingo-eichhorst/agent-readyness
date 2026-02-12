// Package c3 analyzes C3 (Architecture) metrics for agent-readiness.
//
// C3 measures architectural complexity for Python codebases using Tree-sitter.
//
// Python-specific architectural patterns:
// - Imports use dotted module paths (e.g., "from pkg.sub import foo")
// - Relative imports with dots ("from .sibling import bar", "from ..parent import baz")
// - __init__.py files define package boundaries and can create false circular deps
// - Dynamic imports (importlib, __import__) cannot be statically analyzed
// - site-packages and stdlib imports must be filtered (external dependencies)
// - __all__ list defines public API (unlike TypeScript's explicit export keyword)
//
// This file handles Python's import semantics and package structure for dependency
// analysis and dead code detection.
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

// pyFilterSourceFiles filters to source-only Python files (not test files).
func pyFilterSourceFiles(files []*parser.ParsedTreeSitterFile) []*parser.ParsedTreeSitterFile {
	var result []*parser.ParsedTreeSitterFile
	for _, f := range files {
		if isTestFileByPath(f.RelPath) {
			continue
		}
		result = append(result, f)
	}
	return result
}

// isTestFileByPath checks if a file path indicates a test file.
func isTestFileByPath(path string) bool {
	base := strings.ToLower(path)
	parts := strings.Split(base, string(os.PathSeparator))
	if len(parts) > 0 {
		base = parts[len(parts)-1]
	}
	slashParts := strings.Split(base, "/")
	if len(slashParts) > 0 {
		base = slashParts[len(slashParts)-1]
	}

	return strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(base, "_test.py") ||
		base == "conftest.py"
}

// pyBuildImportGraph builds an import graph from Python files.
//
// It tracks intra-project imports only (skips stdlib/third-party).
//
// Python-specific import edge cases:
// - Relative imports (from . import foo) require resolving dots to parent packages
// - Deferred imports (import inside function) still create dependencies for analysis
// - __init__.py imports can create false positives (package re-exports)
// - "from X import *" creates dependencies even without explicit names listed
//
// Circular dependency detection via Tarjan's SCC algorithm (see shared package).
// Python's import system technically supports cycles via deferred imports, but
// they still indicate poor architecture that confuses agents trying to understand
// module boundaries and dependency flow.
func pyBuildImportGraph(files []*parser.ParsedTreeSitterFile) *shared.ImportGraph {
	g := &shared.ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	// Build set of known project files (by module name derived from path)
	knownModules := make(map[string]string) // module name -> file relpath
	for _, f := range files {
		modName := pyFileToModule(f.RelPath)
		knownModules[modName] = f.RelPath
	}

	for _, f := range files {
		root := f.Tree.RootNode()
		fromModule := pyFileToModule(f.RelPath)

		shared.WalkTree(root, func(node *tree_sitter.Node) {
			kind := node.Kind()

			switch kind {
			case "import_statement":
				// import foo, import foo.bar
				for i := uint(0); i < node.ChildCount(); i++ {
					child := node.Child(i)
					if child != nil && (child.Kind() == "dotted_name" || child.Kind() == "aliased_import") {
						var modName string
						if child.Kind() == "aliased_import" {
							nameNode := child.ChildByFieldName("name")
							if nameNode != nil {
								modName = shared.NodeText(nameNode, f.Content)
							}
						} else {
							modName = shared.NodeText(child, f.Content)
						}
						if modName != "" && modName != fromModule {
							if _, ok := knownModules[modName]; ok {
								g.Forward[fromModule] = appendUnique(g.Forward[fromModule], modName)
								g.Reverse[modName] = appendUnique(g.Reverse[modName], fromModule)
							}
						}
					}
				}

			case "import_from_statement":
				// from foo import bar
				modNode := node.ChildByFieldName("module_name")
				if modNode == nil {
					// Try alternative field name
					for i := uint(0); i < node.ChildCount(); i++ {
						child := node.Child(i)
						if child != nil && (child.Kind() == "dotted_name" || child.Kind() == "relative_import") {
							modNode = child
							break
						}
					}
				}
				if modNode != nil {
					modName := shared.NodeText(modNode, f.Content)
					// Handle relative imports (starting with .)
					if strings.HasPrefix(modName, ".") {
						modName = pyResolveRelativeImport(fromModule, modName)
					}
					if modName != "" && modName != fromModule {
						if _, ok := knownModules[modName]; ok {
							g.Forward[fromModule] = appendUnique(g.Forward[fromModule], modName)
							g.Reverse[modName] = appendUnique(g.Reverse[modName], fromModule)
						}
					}
				}
			}
		})
	}

	return g
}

// pyFileToModule converts a file relative path to a Python module name.
// e.g., "utils.py" -> "utils", "pkg/sub/foo.py" -> "pkg.sub.foo"
func pyFileToModule(relPath string) string {
	// Remove .py extension
	name := strings.TrimSuffix(relPath, ".py")
	name = strings.TrimSuffix(name, ".pyi")
	// Replace path separators with dots
	name = strings.ReplaceAll(name, string(os.PathSeparator), ".")
	name = strings.ReplaceAll(name, "/", ".")
	// Remove __init__ suffix
	name = strings.TrimSuffix(name, ".__init__")
	return name
}

// pyResolveRelativeImport resolves a relative import to an absolute module name.
func pyResolveRelativeImport(fromModule, relImport string) string {
	dots := 0
	for _, c := range relImport {
		if c == '.' {
			dots++
		} else {
			break
		}
	}

	parts := strings.Split(fromModule, ".")
	// Go up 'dots' levels (first dot = current package)
	if dots > len(parts) {
		return ""
	}
	base := strings.Join(parts[:len(parts)-(dots-1)], ".")
	rest := relImport[dots:]
	if rest == "" {
		return base
	}
	if base == "" {
		return rest
	}
	return base + "." + rest
}

// pyDetectDeadCode finds top-level functions and classes not imported by other files.
//
// Why dead code matters for agents: Unused exports inflate the API surface agents
// must parse and understand. Dead code also creates false leads during debugging
// ("this function looks relevant but isn't actually called anywhere"). Pruning
// dead exports focuses agent attention on the active codebase.
//
// Python-specific considerations:
// - __all__ list defines public API (if present, only those names are "public")
// - Private names (starting with _) are excluded by convention
// - Decorated definitions (@decorator) are unwrapped to find the actual def/class
//
// Limitation: Jupyter notebooks cannot be analyzed (dynamic execution model).
func pyDetectDeadCode(files []*parser.ParsedTreeSitterFile) []types.DeadExport {
	if len(files) <= 1 {
		return nil // Single file: no cross-file analysis possible
	}

	// Collect all top-level definitions
	type definition struct {
		name string
		file string
		line int
		kind string
	}

	var defs []definition
	for _, f := range files {
		if isTestFileByPath(f.RelPath) {
			continue
		}
		root := f.Tree.RootNode()
		for i := uint(0); i < root.ChildCount(); i++ {
			child := root.Child(i)
			if child == nil {
				continue
			}

			kind := child.Kind()
			var nameNode *tree_sitter.Node
			var defKind string

			switch kind {
			case "function_definition":
				nameNode = child.ChildByFieldName("name")
				defKind = "func"
			case "class_definition":
				nameNode = child.ChildByFieldName("name")
				defKind = "type"
			case "decorated_definition":
				// Unwrap to find inner definition
				for j := uint(0); j < child.ChildCount(); j++ {
					inner := child.Child(j)
					if inner == nil {
						continue
					}
					switch inner.Kind() {
					case "function_definition":
						nameNode = inner.ChildByFieldName("name")
						defKind = "func"
					case "class_definition":
						nameNode = inner.ChildByFieldName("name")
						defKind = "type"
					}
				}
			}

			if nameNode == nil {
				continue
			}

			name := shared.NodeText(nameNode, f.Content)
			// Skip private names
			if strings.HasPrefix(name, "_") {
				continue
			}

			defs = append(defs, definition{
				name: name,
				file: f.RelPath,
				line: int(nameNode.StartPosition().Row) + 1,
				kind: defKind,
			})
		}
	}

	// Collect all imported names across files
	importedNames := make(map[string]bool)
	for _, f := range files {
		root := f.Tree.RootNode()
		shared.WalkTree(root, func(node *tree_sitter.Node) {
			if node.Kind() != "import_from_statement" {
				return
			}
			// Collect imported names
			for i := uint(0); i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child == nil {
					continue
				}
				switch child.Kind() {
				case "aliased_import":
					nameNode := child.ChildByFieldName("name")
					if nameNode != nil {
						importedNames[shared.NodeText(nameNode, f.Content)] = true
					}
				case "dotted_name":
					// This might be the module name, not an imported name
					// Only count if after "import" keyword in the statement
				case "identifier":
					// Check if this identifier is part of the import list
					parent := child.Parent()
					if parent != nil && parent.Kind() == "import_from_statement" {
						name := shared.NodeText(child, f.Content)
						if name != "import" && name != "from" && name != "as" {
							importedNames[name] = true
						}
					}
				}
			}
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

// pyAnalyzeDirectoryDepth computes max and average directory depth from Python file paths.
func pyAnalyzeDirectoryDepth(files []*parser.ParsedTreeSitterFile, rootDir string) (int, float64) {
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
		depth := strings.Count(relPath, string(os.PathSeparator))
		// Also count forward slashes
		depth += strings.Count(relPath, "/") - strings.Count(relPath, string(os.PathSeparator))
		if depth < 0 {
			depth = 0
		}

		totalDepth += depth
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	avg := float64(totalDepth) / float64(len(files))
	return maxDepth, avg
}

// appendUnique appends s to slice only if not already present.
func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}
