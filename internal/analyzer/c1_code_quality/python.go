package c1

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// pyDupConfig is the language-specific configuration for Python duplication detection.
var pyDupConfig = langDupConfig{
	blockKinds: []string{"block", "module"},
	skipKinds:  []string{"comment", "newline", ""},
	callKind:   "call",
	assignKind: "assignment",
}

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
//
// Python-specific handling:
// - className tracks the enclosing class for method naming (Class.method format)
// - Handles decorated_definition nodes (functions with @decorators)
// - Processes nested functions and classes within function bodies
// - Computes line count from Tree-sitter node start/end positions
func pyWalkFunctions(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	if node == nil {
		return
	}

	switch node.Kind() {
	case "class_definition":
		pyWalkClassBody(node, content, file, results)
		return

	case "decorated_definition":
		pyWalkDecoratedDef(node, content, file, className, results)
		return

	case "function_definition":
		pyRecordFunction(node, content, file, className, results)
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

// pyWalkClassBody walks a class definition's body, passing the class name to child walkers.
func pyWalkClassBody(node *tree_sitter.Node, content []byte, file string, results *[]types.FunctionMetric) {
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
}

// pyWalkDecoratedDef unwraps a decorated_definition to find inner function/class definitions.
func pyWalkDecoratedDef(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			childKind := child.Kind()
			if childKind == "function_definition" || childKind == "class_definition" {
				pyWalkFunctions(child, content, file, className, results)
			}
		}
	}
}

// pyRecordFunction records a function metric and walks the body for nested definitions.
func pyRecordFunction(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	name := pyExtractFuncName(node, content, className)
	startRow := int(node.StartPosition().Row)
	endRow := int(node.EndPosition().Row)

	*results = append(*results, types.FunctionMetric{
		Name:       name,
		File:       file,
		Line:       startRow + 1,
		Complexity: pyComputeComplexity(node),
		LineCount:  endRow - startRow + 1,
	})

	body := node.ChildByFieldName("body")
	if body != nil {
		pyWalkFunctionsInBody(body, content, file, className, results)
	}
}

// pyExtractFuncName extracts the function name, prefixed with className if inside a class.
func pyExtractFuncName(node *tree_sitter.Node, content []byte, className string) string {
	return extractClassedFuncName(node, content, className)
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
//
// Complexity calculation:
// - Base complexity is 1 (single execution path)
// - Each branching construct adds 1: if/elif, for, while, except, and, or
// - Boolean operators (and/or) in conditions add branches (short-circuit evaluation)
// - Nested function definitions are excluded from parent's complexity count
//
// This matches the standard McCabe complexity metric used by tools like radon and pylint.
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

