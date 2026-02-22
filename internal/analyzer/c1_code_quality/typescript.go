package c1

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// tsDupConfig is the language-specific configuration for TypeScript duplication detection.
var tsDupConfig = langDupConfig{
	blockKinds: []string{"statement_block", "program"},
	skipKinds:  []string{"comment", "", "{", "}"},
	callKind:   "call_expression",
	assignKind: "assignment_expression",
}


// tsAnalyzeFunctions extracts per-function complexity and line count from TypeScript files.
//
// TypeScript-specific handling:
// - Processes function_declaration, method_definition, and arrow_function nodes
// - Tracks className for method naming (Class.method format)
// - Handles anonymous arrow functions (assigns synthetic names based on context)
// - Computes line count from Tree-sitter node start/end positions
func tsAnalyzeFunctions(files []*parser.ParsedTreeSitterFile) []types.FunctionMetric {
	var results []types.FunctionMetric

	for _, f := range files {
		root := f.Tree.RootNode()
		tsWalkFunctions(root, f.Content, f.RelPath, "", &results)
	}

	return results
}

// tsWalkFunctions recursively walks the AST to find function declarations, arrow functions, and methods.
//
// TypeScript function types handled:
// - function_declaration: Named function declarations (function foo() {})
// - method_definition: Class methods and object method shorthand
// - arrow_function: Arrow functions (const f = () => {})
// - Tracks enclosing class name for proper method identification
//
// Returns early for function nodes to avoid counting nested functions multiple times.
func tsWalkFunctions(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	if node == nil {
		return
	}

	kind := node.Kind()

	switch kind {
	case "class_declaration":
		tsWalkClassBody(node, content, file, results)
		return

	case "function_declaration", "method_definition":
		name := tsExtractFuncName(node, content, className)
		*results = append(*results, tsBuildFuncMetric(node, content, file, name))
		return

	case "arrow_function":
		name := tsArrowFunctionName(node, content)
		*results = append(*results, tsBuildFuncMetric(node, content, file, name))
		return
	}

	// Default: recurse into children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			tsWalkFunctions(child, content, file, className, results)
		}
	}
}

// tsWalkClassBody walks a class declaration's body, passing the class name to child walkers.
func tsWalkClassBody(node *tree_sitter.Node, content []byte, file string, results *[]types.FunctionMetric) {
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
				tsWalkFunctions(child, content, file, clsName, results)
			}
		}
	}
}

// tsExtractFuncName extracts the function name, prefixed with className if inside a class.
func tsExtractFuncName(node *tree_sitter.Node, content []byte, className string) string {
	return extractClassedFuncName(node, content, className)
}

// tsBuildFuncMetric creates a FunctionMetric from a function/method/arrow AST node.
func tsBuildFuncMetric(node *tree_sitter.Node, content []byte, file string, name string) types.FunctionMetric {
	startRow := int(node.StartPosition().Row)
	endRow := int(node.EndPosition().Row)
	return types.FunctionMetric{
		Name:       name,
		File:       file,
		Line:       startRow + 1,
		Complexity: tsComputeComplexity(node, content),
		LineCount:  endRow - startRow + 1,
	}
}

// tsArrowFunctionName tries to extract the name of an arrow function from its parent variable declarator.
func tsArrowFunctionName(node *tree_sitter.Node, content []byte) string {
	parent := node.Parent()
	if parent != nil && parent.Kind() == "variable_declarator" {
		nameNode := parent.ChildByFieldName("name")
		if nameNode != nil {
			return shared.NodeText(nameNode, content)
		}
	}
	return "<anonymous>"
}

// tsComputeComplexity computes McCabe cyclomatic complexity for a TypeScript function.
//
// Complexity calculation:
// - Base complexity is 1 (single execution path)
// - Each branching construct adds 1: if, for, while, switch case, catch, ternary
// - Boolean operators (&&, ||, ??) in expressions add branches (short-circuit evaluation)
// - Nested function/arrow definitions are excluded from parent's complexity count
//
// This matches the standard McCabe complexity metric used by tools like ESLint.
//
// Why complexity matters for AI agents: High complexity (>15) requires multi-step
// reasoning across many execution paths. Agents struggle to track all branches
// simultaneously, leading to bugs where edge cases are missed. Functions with
// complexity >20 have exponentially higher agent error rates. Threshold: keep
// complexity ≤10 for agent-friendly code (single-digit branch count).
func tsComputeComplexity(funcNode *tree_sitter.Node, content []byte) int {
	body := funcNode.ChildByFieldName("body")
	if body == nil {
		return 1
	}

	complexity := 1
	tsWalkForComplexity(body, content, &complexity)
	return complexity
}

// tsWalkForComplexity recursively walks AST nodes counting branching constructs.
func tsWalkForComplexity(node *tree_sitter.Node, content []byte, complexity *int) {
	if node == nil {
		return
	}

	kind := node.Kind()

	// Skip nested function definitions to avoid double-counting
	if tsIsNestedFunction(kind) {
		return
	}

	tsIncrementComplexityForNode(node, kind, content, complexity)

	// Recurse into children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			tsWalkForComplexity(child, content, complexity)
		}
	}
}

// tsIsNestedFunction checks if node kind is a nested function definition.
func tsIsNestedFunction(kind string) bool {
	return kind == "function_declaration" || kind == "arrow_function" || kind == "function_expression"
}

// tsIncrementComplexityForNode increments complexity based on node type.
func tsIncrementComplexityForNode(node *tree_sitter.Node, kind string, content []byte, complexity *int) {
	switch kind {
	case "if_statement", "catch_clause", "ternary_expression":
		*complexity++
	case "for_statement", "for_in_statement":
		*complexity++
	case "while_statement", "do_statement":
		*complexity++
	case "switch_case":
		if tsIsNonDefaultCase(node, content) {
			*complexity++
		}
	case "binary_expression":
		if tsIsLogicalOperator(node, content) {
			*complexity++
		}
	}
}

// tsIsNonDefaultCase checks if a switch_case is not the default case.
func tsIsNonDefaultCase(node *tree_sitter.Node, content []byte) bool {
	if node.ChildCount() == 0 {
		return false
	}
	firstChild := node.Child(0)
	return firstChild != nil && shared.NodeText(firstChild, content) != "default"
}

// tsIsLogicalOperator checks if a binary expression uses a logical operator.
func tsIsLogicalOperator(node *tree_sitter.Node, content []byte) bool {
	opNode := node.ChildByFieldName("operator")
	if opNode == nil {
		return false
	}
	op := shared.NodeText(opNode, content)
	return op == "&&" || op == "||" || op == "??"
}

// tsAnalyzeFileSizes computes file size metrics for TypeScript files.
