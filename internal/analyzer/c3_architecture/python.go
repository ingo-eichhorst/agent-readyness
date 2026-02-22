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
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

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
	return buildImportGraph(files, []string{"import_statement", "import_from_statement"}, pyImportExtractor{})
}

// pyImportExtractor implements importExtractor for Python source files.
type pyImportExtractor struct{}

// fileToModule converts a Python file rel path to its dotted module name.
func (pyImportExtractor) fileToModule(filePath string) string {
	return pyFileToModule(filePath)
}

// extractImports returns intra-project module targets from Python import AST nodes.
func (pyImportExtractor) extractImports(node *tree_sitter.Node, content []byte, fromModule string, knownModules map[string]string) []string {
	switch node.Kind() {
	case "import_statement":
		return pyExtractImportStatement(node, content, knownModules)
	case "import_from_statement":
		return pyExtractImportFromStatement(node, content, fromModule, knownModules)
	}
	return nil
}

// pyExtractImportStatement extracts module names from "import foo" / "import foo.bar" nodes.
func pyExtractImportStatement(node *tree_sitter.Node, content []byte, knownModules map[string]string) []string {
	var targets []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() != "dotted_name" && child.Kind() != "aliased_import" {
			continue
		}
		var modName string
		if child.Kind() == "aliased_import" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				modName = shared.NodeText(nameNode, content)
			}
		} else {
			modName = shared.NodeText(child, content)
		}
		if modName != "" {
			if _, ok := knownModules[modName]; ok {
				targets = append(targets, modName)
			}
		}
	}
	return targets
}

// pyExtractImportFromStatement extracts module names from "from foo import bar" nodes.
func pyExtractImportFromStatement(node *tree_sitter.Node, content []byte, fromModule string, knownModules map[string]string) []string {
	modNode := node.ChildByFieldName("module_name")
	if modNode == nil {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == "dotted_name" || child.Kind() == "relative_import") {
				modNode = child
				break
			}
		}
	}
	if modNode == nil {
		return nil
	}
	modName := shared.NodeText(modNode, content)
	if strings.HasPrefix(modName, ".") {
		modName = pyResolveRelativeImport(fromModule, modName)
	}
	if modName == "" {
		return nil
	}
	if _, ok := knownModules[modName]; ok {
		return []string{modName}
	}
	return nil
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
// pyDeadCodeExtractor implements deadCodeExtractor for Python files.
type pyDeadCodeExtractor struct{}

func pyDetectDeadCode(files []*parser.ParsedTreeSitterFile) []types.DeadExport {
	return detectTreeSitterDeadCode(files, pyDeadCodeExtractor{})
}

func (e pyDeadCodeExtractor) collectDefinitions(files []*parser.ParsedTreeSitterFile) []codeDefinition {
	return pyCollectTopLevelDefs(files)
}

func (e pyDeadCodeExtractor) collectImportedNames(files []*parser.ParsedTreeSitterFile) map[string]bool {
	return pyCollectImportedNames(files)
}

// pyCollectTopLevelDefs collects all public top-level function and class definitions.
func pyCollectTopLevelDefs(files []*parser.ParsedTreeSitterFile) []codeDefinition {
	var defs []codeDefinition
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
			if d, ok := pyExtractDefinition(child, f.Content, f.RelPath); ok {
				defs = append(defs, d)
			}
		}
	}
	return defs
}

// pyExtractDefinition extracts a definition from a top-level AST node if it is a public function or class.
func pyExtractDefinition(node *tree_sitter.Node, content []byte, relPath string) (codeDefinition, bool) {
	kind := node.Kind()
	var nameNode *tree_sitter.Node
	var defKind string

	switch kind {
	case "function_definition":
		nameNode = node.ChildByFieldName("name")
		defKind = "func"
	case "class_definition":
		nameNode = node.ChildByFieldName("name")
		defKind = "type"
	case "decorated_definition":
		nameNode, defKind = pyUnwrapDecoratedDef(node)
	}

	if nameNode == nil {
		return codeDefinition{}, false
	}

	name := shared.NodeText(nameNode, content)
	if strings.HasPrefix(name, "_") {
		return codeDefinition{}, false
	}

	return codeDefinition{
		Name: name,
		File: relPath,
		Line: int(nameNode.StartPosition().Row) + 1,
		Kind: defKind,
	}, true
}

// pyUnwrapDecoratedDef unwraps a decorated_definition to find the inner function or class name node.
func pyUnwrapDecoratedDef(node *tree_sitter.Node) (*tree_sitter.Node, string) {
	for j := uint(0); j < node.ChildCount(); j++ {
		inner := node.Child(j)
		if inner == nil {
			continue
		}
		switch inner.Kind() {
		case "function_definition":
			return inner.ChildByFieldName("name"), "func"
		case "class_definition":
			return inner.ChildByFieldName("name"), "type"
		}
	}
	return nil, ""
}

// pyCollectImportedNames gathers all names imported via import_from_statement across files.
func pyCollectImportedNames(files []*parser.ParsedTreeSitterFile) map[string]bool {
	importedNames := make(map[string]bool)
	for _, f := range files {
		root := f.Tree.RootNode()
		shared.WalkTree(root, func(node *tree_sitter.Node) {
			if node.Kind() != "import_from_statement" {
				return
			}
			pyCollectNamesFromImport(node, f.Content, importedNames)
		})
	}
	return importedNames
}

// pyCollectNamesFromImport extracts imported symbol names from a single import_from_statement.
func pyCollectNamesFromImport(node *tree_sitter.Node, content []byte, names map[string]bool) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "aliased_import":
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				names[shared.NodeText(nameNode, content)] = true
			}
		case "dotted_name":
			// Module name, not an imported name
		case "identifier":
			parent := child.Parent()
			if parent != nil && parent.Kind() == "import_from_statement" {
				name := shared.NodeText(child, content)
				if name != "import" && name != "from" && name != "as" {
					names[name] = true
				}
			}
		}
	}
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
