package c3

import (
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/analyzer"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
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
// It tracks intra-project imports only (skips stdlib/third-party).
func pyBuildImportGraph(files []*parser.ParsedTreeSitterFile) *analyzer.ImportGraph {
	g := &analyzer.ImportGraph{
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

		analyzer.WalkTree(root, func(node *tree_sitter.Node) {
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
								modName = analyzer.NodeText(nameNode, f.Content)
							}
						} else {
							modName = analyzer.NodeText(child, f.Content)
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
					modName := analyzer.NodeText(modNode, f.Content)
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

			name := analyzer.NodeText(nameNode, f.Content)
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
		analyzer.WalkTree(root, func(node *tree_sitter.Node) {
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
						importedNames[analyzer.NodeText(nameNode, f.Content)] = true
					}
				case "dotted_name":
					// This might be the module name, not an imported name
					// Only count if after "import" keyword in the statement
				case "identifier":
					// Check if this identifier is part of the import list
					parent := child.Parent()
					if parent != nil && parent.Kind() == "import_from_statement" {
						name := analyzer.NodeText(child, f.Content)
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
