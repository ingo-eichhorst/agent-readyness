package c3

import (
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// tsBuildImportGraph builds an import graph from TypeScript files.
// It tracks intra-project imports only (skips node_modules/third-party).
func tsBuildImportGraph(files []*parser.ParsedTreeSitterFile) *shared.ImportGraph {
	g := &shared.ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	// Build set of known project files by their relative path (without extension)
	knownFiles := make(map[string]string) // normalized path -> original relPath
	for _, f := range files {
		normalized := tsNormalizePath(f.RelPath)
		knownFiles[normalized] = f.RelPath
	}

	for _, f := range files {
		root := f.Tree.RootNode()
		fromFile := tsNormalizePath(f.RelPath)
		fromDir := filepath.Dir(f.RelPath)

		shared.WalkTree(root, func(node *tree_sitter.Node) {
			kind := node.Kind()

			var modulePath string

			switch kind {
			case "import_statement":
				// ESM: import { foo } from "./bar"
				src := node.ChildByFieldName("source")
				if src != nil {
					modulePath = shared.TsStripQuotes(shared.NodeText(src, f.Content))
				}

			case "call_expression":
				// CommonJS: require("./bar")
				fn := node.ChildByFieldName("function")
				if fn == nil {
					return
				}
				if shared.NodeText(fn, f.Content) != "require" {
					return
				}
				args := node.ChildByFieldName("arguments")
				if args == nil {
					return
				}
				// Get first argument (string)
				for i := uint(0); i < args.ChildCount(); i++ {
					child := args.Child(i)
					if child != nil && child.Kind() == "string" {
						modulePath = shared.TsStripQuotes(shared.NodeText(child, f.Content))
						break
					}
				}

			default:
				return
			}

			if modulePath == "" {
				return
			}

			// Only track relative imports (intra-project)
			if !strings.HasPrefix(modulePath, ".") {
				return
			}

			// Resolve relative path
			resolved := filepath.Join(fromDir, modulePath)
			resolved = filepath.Clean(resolved)
			// Normalize separators
			resolved = strings.ReplaceAll(resolved, string(os.PathSeparator), "/")

			// Try to match against known files
			normalizedResolved := tsNormalizePath(resolved)
			if _, ok := knownFiles[normalizedResolved]; ok {
				if normalizedResolved != fromFile {
					g.Forward[fromFile] = appendUnique(g.Forward[fromFile], normalizedResolved)
					g.Reverse[normalizedResolved] = appendUnique(g.Reverse[normalizedResolved], fromFile)
				}
			}
		})
	}

	return g
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

// tsDetectDeadCode finds exported symbols not imported by other project files.
func tsDetectDeadCode(files []*parser.ParsedTreeSitterFile) []types.DeadExport {
	if len(files) <= 1 {
		return nil // Single file: no cross-file analysis possible
	}

	var defs []tsExportDef

	for _, f := range files {
		if shared.TsIsTestFile(f.RelPath) {
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
