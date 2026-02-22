package c3

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
)

// importExtractor is implemented by each language to extract import edges from AST nodes.
type importExtractor interface {
	// extractImports returns the resolved module/file identifiers being imported by
	// the given AST node. Only intra-project imports should be returned; third-party
	// and stdlib imports must be filtered by the implementation. fromFile is the
	// normalized module/path identifier for the file being parsed (used for relative
	// import resolution). knownModules maps known module identifiers to their rel paths.
	extractImports(node *tree_sitter.Node, content []byte, fromFile string, knownModules map[string]string) []string

	// fileToModule converts a file's RelPath to the module/path identifier used as
	// the graph node key (e.g. dotted Python module name, or normalized TS path).
	fileToModule(filePath string) string
}

// buildImportGraph builds a forward/reverse import graph from parsed Tree-sitter files.
// It delegates language-specific AST interpretation to the provided importExtractor.
// importNodeTypes lists the AST node kinds to visit (e.g. "import_statement"); pass nil
// to visit every node (TypeScript needs call_expression for require() calls).
func buildImportGraph(files []*parser.ParsedTreeSitterFile, importNodeTypes []string, extractor importExtractor) *shared.ImportGraph {
	graph := &shared.ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	// Build map of all known module identifiers so extractors can filter to intra-project.
	knownModules := make(map[string]string)
	for _, f := range files {
		knownModules[extractor.fileToModule(f.RelPath)] = f.RelPath
	}

	nodeTypeSet := make(map[string]bool, len(importNodeTypes))
	for _, t := range importNodeTypes {
		nodeTypeSet[t] = true
	}
	filterByType := len(importNodeTypes) > 0

	for _, f := range files {
		root := f.Tree.RootNode()
		fromModule := extractor.fileToModule(f.RelPath)

		shared.WalkTree(root, func(node *tree_sitter.Node) {
			if filterByType && !nodeTypeSet[node.Kind()] {
				return
			}
			targets := extractor.extractImports(node, f.Content, fromModule, knownModules)
			for _, to := range targets {
				if to == "" || to == fromModule {
					continue
				}
				graph.Forward[fromModule] = appendUnique(graph.Forward[fromModule], to)
				graph.Reverse[to] = appendUnique(graph.Reverse[to], fromModule)
			}
		})
	}

	return graph
}
