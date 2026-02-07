package c1

import (
	"fmt"
	"hash"
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// pyAnalyzeFunctions extracts per-function complexity and line count from Python files.
// It walks Tree-sitter ASTs to find function_definition nodes and computes
// cyclomatic complexity by counting branches in the function body.
func pyAnalyzeFunctions(files []*parser.ParsedTreeSitterFile) []types.FunctionMetric {
	var results []types.FunctionMetric

	for _, f := range files {
		root := f.Tree.RootNode()
		pyWalkFunctions(root, f.Content, f.RelPath, "", &results)
	}

	return results
}

// pyWalkFunctions recursively walks the AST to find function definitions.
// className tracks the enclosing class for method naming.
func pyWalkFunctions(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	if node == nil {
		return
	}

	kind := node.Kind()

	if kind == "class_definition" {
		nameNode := node.ChildByFieldName("name")
		clsName := ""
		if nameNode != nil {
			clsName = shared.NodeText(nameNode, content)
		}
		body := node.ChildByFieldName("body")
		if body != nil {
			for i := uint(0); i < body.ChildCount(); i++ {
				child := body.Child(i)
				if child != nil {
					pyWalkFunctions(child, content, file, clsName, results)
				}
			}
		}
		return
	}

	// Handle decorated_definition: unwrap to inner function/class
	if kind == "decorated_definition" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil {
				childKind := child.Kind()
				if childKind == "function_definition" || childKind == "class_definition" {
					pyWalkFunctions(child, content, file, className, results)
				}
			}
		}
		return
	}

	if kind == "function_definition" {
		nameNode := node.ChildByFieldName("name")
		name := ""
		if nameNode != nil {
			name = shared.NodeText(nameNode, content)
		}

		if className != "" {
			name = className + "." + name
		}

		startRow := int(node.StartPosition().Row)
		endRow := int(node.EndPosition().Row)
		lineCount := endRow - startRow + 1

		complexity := pyComputeComplexity(node)

		*results = append(*results, types.FunctionMetric{
			Name:       name,
			File:       file,
			Line:       startRow + 1,
			Complexity: complexity,
			LineCount:  lineCount,
		})

		// Walk body for nested function/class definitions only
		body := node.ChildByFieldName("body")
		if body != nil {
			pyWalkFunctionsInBody(body, content, file, className, results)
		}
		return
	}

	// Default: recurse into children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			pyWalkFunctions(child, content, file, className, results)
		}
	}
}

// pyWalkFunctionsInBody finds nested function/class definitions inside a function body.
func pyWalkFunctionsInBody(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	if node == nil {
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "function_definition" || kind == "class_definition" || kind == "decorated_definition" {
			pyWalkFunctions(child, content, file, className, results)
		} else {
			pyWalkFunctionsInBody(child, content, file, className, results)
		}
	}
}

// pyComputeComplexity computes McCabe cyclomatic complexity for a Python function.
// Base complexity is 1. Each branching construct adds 1.
// Nested function definitions are excluded.
func pyComputeComplexity(funcNode *tree_sitter.Node) int {
	complexity := 1
	body := funcNode.ChildByFieldName("body")
	if body == nil {
		return complexity
	}

	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if n == nil {
			return
		}

		kind := n.Kind()

		// Skip nested function/class definitions
		if kind == "function_definition" || kind == "class_definition" {
			return
		}

		switch kind {
		case "if_statement", "elif_clause",
			"for_statement", "while_statement",
			"except_clause", "case_clause",
			"conditional_expression":
			complexity++
		case "boolean_operator":
			complexity++
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}

	walk(body)
	return complexity
}

// pyAnalyzeFileSizes computes file size metrics for Python files.
func pyAnalyzeFileSizes(files []*parser.ParsedTreeSitterFile) types.MetricSummary {
	return analyzeTSFileSizes(files)
}
// pyAnalyzeDuplication detects duplicate code blocks in Python using statement-sequence hashing.
func pyAnalyzeDuplication(files []*parser.ParsedTreeSitterFile) ([]types.DuplicateBlock, float64) {
	cfg := dupConfig{
		blockKinds: []string{"block", "module"},
		skipKinds:  []string{"comment", "newline", ""},
		hashNode:   pyHashNodeStructure,
	}
	return analyzeTSDuplication(files, cfg)
}

// pyHashNodeStructure writes a structural representation of a Python AST node to the hasher.
func pyHashNodeStructure(h hash.Hash64, node *tree_sitter.Node, depth int) {
	if node == nil || depth > 5 {
		return
	}

	kind := node.Kind()
	fmt.Fprint(h, kind)

	childCount := node.ChildCount()
	fmt.Fprintf(h, ":%d", childCount)

	switch kind {
	case "call":
		fn := node.ChildByFieldName("function")
		if fn != nil {
			fmt.Fprint(h, fn.Kind())
		}
	case "assignment":
		fmt.Fprint(h, "=")
	case "return_statement":
		fmt.Fprint(h, "ret")
	}

	for i := uint(0); i < childCount && i < 10; i++ {
		child := node.Child(i)
		if child != nil {
			pyHashNodeStructure(h, child, depth+1)
		}
	}
}

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
