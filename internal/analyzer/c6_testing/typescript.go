package c6

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// tsDetectTests detects test functions in TypeScript files and counts test vs source files.
// Returns test function metrics, test file count, and source file count.
func tsDetectTests(files []*parser.ParsedTreeSitterFile) ([]types.TestFunctionMetric, int, int) {
	var testFuncs []types.TestFunctionMetric
	testFileCount := 0
	srcFileCount := 0

	for _, f := range files {
		isTest := shared.TsIsTestFile(f.RelPath)
		if isTest {
			testFileCount++
		} else {
			srcFileCount++
		}

		if !isTest {
			continue
		}

		// Walk test file for describe/it/test calls
		root := f.Tree.RootNode()
		tsCollectTestFunctions(root, f.Content, f.RelPath, &testFuncs)
	}

	return testFuncs, testFileCount, srcFileCount
}

// tsCollectTestFunctions walks the AST collecting test functions (it/test calls).
func tsCollectTestFunctions(node *tree_sitter.Node, content []byte, relPath string, results *[]types.TestFunctionMetric) {
	if node == nil {
		return
	}

	kind := node.Kind()

	if kind == "call_expression" {
		fn := node.ChildByFieldName("function")
		if fn != nil {
			fnName := shared.NodeText(fn, content)

			// it("...", () => { ... }) or test("...", () => { ... })
			if fnName == "it" || fnName == "test" {
				testName := tsExtractFirstStringArg(node, content)
				if testName == "" {
					testName = fnName
				}

				// Count assertions in the test body
				assertCount := 0
				args := node.ChildByFieldName("arguments")
				if args != nil {
					// Find the callback (arrow function or function expression)
					for i := uint(0); i < args.ChildCount(); i++ {
						child := args.Child(i)
						if child != nil && (child.Kind() == "arrow_function" || child.Kind() == "function_expression") {
							assertCount = tsCountAssertions(child, content)
							break
						}
					}
				}

				*results = append(*results, types.TestFunctionMetric{
					Name:           testName,
					File:           relPath,
					Line:           int(node.StartPosition().Row) + 1,
					AssertionCount: assertCount,
				})
			}
		}
	}

	// Recurse into children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			tsCollectTestFunctions(child, content, relPath, results)
		}
	}
}

// tsExtractFirstStringArg extracts the first string argument from a call expression.
func tsExtractFirstStringArg(callNode *tree_sitter.Node, content []byte) string {
	args := callNode.ChildByFieldName("arguments")
	if args == nil {
		return ""
	}

	for i := uint(0); i < args.ChildCount(); i++ {
		child := args.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "string" || kind == "template_string" {
			text := shared.NodeText(child, content)
			return shared.TsStripQuotes(text)
		}
	}
	return ""
}

// tsCountAssertions counts assertion calls within a TypeScript function body.
// Counts: expect() chains, assert/assert.* calls.
func tsCountAssertions(funcNode *tree_sitter.Node, content []byte) int {
	body := funcNode.ChildByFieldName("body")
	if body == nil {
		return 0
	}

	count := 0
	tsWalkForAssertions(body, funcNode, content, &count)
	return count
}

// tsWalkForAssertions recursively walks nodes counting assertion calls.
func tsWalkForAssertions(node, funcNode *tree_sitter.Node, content []byte, count *int) {
	if node == nil {
		return
	}

	kind := node.Kind()

	// Skip nested function definitions
	if tsIsNestedFunctionNode(node, funcNode, kind) {
		return
	}

	if kind == "call_expression" {
		if tsIsAssertionCall(node, content, count) {
			return // Don't double-count the chain
		}
	}

	// Recurse into children
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			tsWalkForAssertions(child, funcNode, content, count)
		}
	}
}

// tsIsNestedFunctionNode checks if node is a nested function (not the parent).
func tsIsNestedFunctionNode(node, funcNode *tree_sitter.Node, kind string) bool {
	if kind == "arrow_function" && node != funcNode {
		return true
	}
	if kind == "function_expression" && node != funcNode {
		return true
	}
	return false
}

// tsIsAssertionCall checks if a call expression is an assertion and increments count.
func tsIsAssertionCall(node *tree_sitter.Node, content []byte, count *int) bool {
	fn := node.ChildByFieldName("function")
	if fn == nil {
		return false
	}

	fnText := shared.NodeText(fn, content)

	// Jest/Vitest: expect(x).toBe(y)
	if fnText == "expect" || strings.HasPrefix(fnText, "expect(") {
		*count++
		return true
	}

	// Member expression: expect(...).toBe(...)
	if strings.Contains(fnText, "expect(") || strings.Contains(fnText, "expect (") {
		*count++
		return true
	}

	// Node.js assert module
	if fnText == "assert" || strings.HasPrefix(fnText, "assert.") {
		*count++
		return true
	}

	return false
}

// tsExternalDepModules are TypeScript/Node.js packages that indicate external/impure dependencies.
var tsExternalDepModules = map[string]bool{
	"axios":      true,
	"node-fetch": true,
	"pg":         true,
	"mysql2":     true,
	"mongoose":   true,
	"prisma":     true,
	"redis":      true,
	"ioredis":    true,
	"http":       true,
	"https":      true,
	"net":        true,
	"fs":         true,
	"child_process": true,
	"got":        true,
	"superagent": true,
	"knex":       true,
	"typeorm":    true,
	"sequelize":  true,
}

// tsAnalyzeIsolation checks test file imports for external/impure dependencies.
// Returns isolation score (0-100, where 100 = fully isolated).
func tsAnalyzeIsolation(files []*parser.ParsedTreeSitterFile, testFuncs []types.TestFunctionMetric) float64 {
	if len(testFuncs) == 0 {
		return 100 // No tests = vacuously isolated
	}

	testFileHasExtDep := tsCheckTestFilesForExternalDeps(files)
	return tsCalculateIsolationScore(testFuncs, testFileHasExtDep)
}

// tsCheckTestFilesForExternalDeps checks each test file for external dependencies.
func tsCheckTestFilesForExternalDeps(files []*parser.ParsedTreeSitterFile) map[string]bool {
	testFileHasExtDep := make(map[string]bool)

	for _, f := range files {
		if !shared.TsIsTestFile(f.RelPath) {
			continue
		}

		root := f.Tree.RootNode()
		hasExtDep := tsFileHasExternalDep(root, f.Content)
		testFileHasExtDep[f.RelPath] = hasExtDep
	}

	return testFileHasExtDep
}

// tsFileHasExternalDep checks if a file imports any external dependencies.
func tsFileHasExternalDep(root *tree_sitter.Node, content []byte) bool {
	hasExtDep := false

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		if hasExtDep || node.Kind() != "import_statement" {
			return
		}

		src := node.ChildByFieldName("source")
		if src == nil {
			return
		}

		modPath := shared.TsStripQuotes(shared.NodeText(src, content))
		if tsIsExternalDependency(modPath) {
			hasExtDep = true
		}
	})

	return hasExtDep
}

// tsIsExternalDependency checks if a module path is an external dependency.
func tsIsExternalDependency(modPath string) bool {
	// Skip relative imports (intra-project)
	if strings.HasPrefix(modPath, ".") {
		return false
	}

	topLevel := tsExtractTopLevelModule(modPath)
	return tsExternalDepModules[topLevel]
}

// tsExtractTopLevelModule extracts the top-level module name from a module path.
func tsExtractTopLevelModule(modPath string) string {
	idx := strings.Index(modPath, "/")
	if idx <= 0 {
		return modPath
	}

	// Handle scoped packages: @scope/pkg
	if strings.HasPrefix(modPath, "@") {
		parts := strings.SplitN(modPath, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	return modPath[:idx]
}

// tsCalculateIsolationScore counts isolated vs total test functions.
func tsCalculateIsolationScore(testFuncs []types.TestFunctionMetric, testFileHasExtDep map[string]bool) float64 {
	totalTests := len(testFuncs)
	isolatedTests := 0

	for _, tf := range testFuncs {
		if !testFileHasExtDep[tf.File] {
			isolatedTests++
		}
	}

	return float64(isolatedTests) / float64(totalTests) * toPercentC6
}

// tsCountLOC counts test LOC and source LOC for TypeScript files.
func tsCountLOC(files []*parser.ParsedTreeSitterFile) (testLOC, srcLOC int) {
	return countLOCByTestFunc(files, shared.TsIsTestFile)
}
