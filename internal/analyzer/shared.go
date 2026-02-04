// Package analyzer provides code analysis implementations for the ARS pipeline.
// This file re-exports shared utilities from the shared subpackage for backward compatibility.
package analyzer

import (
	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// Type alias for backward compatibility.
type ImportGraph = shared.ImportGraph

// BuildImportGraph re-exports the shared utility.
func BuildImportGraph(pkgs []*parser.ParsedPackage, modulePath string) *ImportGraph {
	return shared.BuildImportGraph(pkgs, modulePath)
}

// WalkTree re-exports the shared utility.
func WalkTree(node *tree_sitter.Node, fn func(*tree_sitter.Node)) {
	shared.WalkTree(node, fn)
}

// NodeText re-exports the shared utility.
func NodeText(node *tree_sitter.Node, content []byte) string {
	return shared.NodeText(node, content)
}

// CountLines re-exports the shared utility.
func CountLines(content []byte) int {
	return shared.CountLines(content)
}

// IsTestFileByPath re-exports the shared utility.
func IsTestFileByPath(path string) bool {
	return shared.IsTestFileByPath(path)
}

// TsIsTestFile re-exports the shared utility.
func TsIsTestFile(path string) bool {
	return shared.TsIsTestFile(path)
}

// TsStripQuotes re-exports the shared utility.
func TsStripQuotes(s string) string {
	return shared.TsStripQuotes(s)
}
