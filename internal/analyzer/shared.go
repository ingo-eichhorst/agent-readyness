// Package analyzer provides code analysis implementations for the ARS pipeline.
// This file re-exports shared utilities from the shared subpackage for backward compatibility.
package analyzer

import (
	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// importGraph is a type alias for backward compatibility.
type importGraph = shared.ImportGraph

// buildImportGraph re-exports the shared utility.
func buildImportGraph(pkgs []*parser.ParsedPackage, modulePath string) *importGraph {
	return shared.BuildImportGraph(pkgs, modulePath)
}

// walkTree re-exports the shared utility.
func walkTree(node *tree_sitter.Node, fn func(*tree_sitter.Node)) {
	shared.WalkTree(node, fn)
}

// nodeText re-exports the shared utility.
func nodeText(node *tree_sitter.Node, content []byte) string {
	return shared.NodeText(node, content)
}

// countLines re-exports the shared utility.
func countLines(content []byte) int {
	return shared.CountLines(content)
}

// isTestFileByPath re-exports the shared utility.
func isTestFileByPath(path string) bool {
	return shared.IsTestFileByPath(path)
}

// tsIsTestFile re-exports the shared utility.
func tsIsTestFile(path string) bool {
	return shared.TsIsTestFile(path)
}

// tsStripQuotes re-exports the shared utility.
func tsStripQuotes(s string) string {
	return shared.TsStripQuotes(s)
}
