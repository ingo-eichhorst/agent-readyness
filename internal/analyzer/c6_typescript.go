package analyzer

import (
	"bytes"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// tsDetectTests detects test functions in TypeScript files and counts test vs source files.
// Returns test function metrics, test file count, and source file count.
func tsDetectTests(files []*parser.ParsedTreeSitterFile) ([]types.TestFunctionMetric, int, int) {
	var testFuncs []types.TestFunctionMetric
	testFileCount := 0
	srcFileCount := 0

	for _, f := range files {
		isTest := tsIsTestFile(f.RelPath)
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
			fnName := nodeText(fn, content)

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
			text := nodeText(child, content)
			return tsStripQuotes(text)
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
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if n == nil {
			return
		}

		kind := n.Kind()

		// Skip nested function definitions
		if kind == "arrow_function" && n != funcNode {
			return
		}
		if kind == "function_expression" && n != funcNode {
			return
		}

		if kind == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				fnText := nodeText(fn, content)

				// Jest/Vitest: expect(x).toBe(y) -- the outer call is .toBe()
				// We count the expect() call as the assertion anchor
				if fnText == "expect" || strings.HasPrefix(fnText, "expect(") {
					count++
					return // Don't double-count the chain
				}

				// Member expression: expect(...).toBe(...)
				// The function text would be like "expect(user.name).toBe"
				if strings.Contains(fnText, "expect(") || strings.Contains(fnText, "expect (") {
					count++
					return
				}

				// Node.js assert module
				if fnText == "assert" || strings.HasPrefix(fnText, "assert.") {
					count++
					return
				}
			}
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}

	walk(body)
	return count
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

	// Check each test file for external imports
	testFileHasExtDep := make(map[string]bool)
	for _, f := range files {
		if !tsIsTestFile(f.RelPath) {
			continue
		}

		root := f.Tree.RootNode()
		hasExtDep := false

		walkTree(root, func(node *tree_sitter.Node) {
			kind := node.Kind()
			if kind != "import_statement" {
				return
			}

			// Extract module path from source
			src := node.ChildByFieldName("source")
			if src == nil {
				return
			}
			modPath := tsStripQuotes(nodeText(src, f.Content))

			// Skip relative imports (intra-project)
			if strings.HasPrefix(modPath, ".") {
				return
			}

			// Check against known external deps
			topLevel := modPath
			if idx := strings.Index(modPath, "/"); idx > 0 {
				// Handle scoped packages: @scope/pkg
				if strings.HasPrefix(modPath, "@") {
					parts := strings.SplitN(modPath, "/", 3)
					if len(parts) >= 2 {
						topLevel = parts[0] + "/" + parts[1]
					}
				} else {
					topLevel = modPath[:idx]
				}
			}

			if tsExternalDepModules[topLevel] {
				hasExtDep = true
			}
		})

		testFileHasExtDep[f.RelPath] = hasExtDep
	}

	// Count isolated vs total test functions
	totalTests := len(testFuncs)
	isolatedTests := 0
	for _, tf := range testFuncs {
		if !testFileHasExtDep[tf.File] {
			isolatedTests++
		}
	}

	return float64(isolatedTests) / float64(totalTests) * 100
}

// tsCountLOC counts test LOC and source LOC for TypeScript files.
func tsCountLOC(files []*parser.ParsedTreeSitterFile) (testLOC, srcLOC int) {
	for _, f := range files {
		lines := bytes.Count(f.Content, []byte("\n")) + 1
		if tsIsTestFile(f.RelPath) {
			testLOC += lines
		} else {
			srcLOC += lines
		}
	}
	return
}

// tsUpdateAssertionDensity recomputes assertion density from all test functions in metrics.
func tsUpdateAssertionDensity(metrics *types.C6Metrics) {
	if len(metrics.TestFunctions) == 0 {
		return
	}

	totalAssertions := 0
	maxAssertions := 0
	maxEntity := ""

	for _, tf := range metrics.TestFunctions {
		totalAssertions += tf.AssertionCount
		if tf.AssertionCount > maxAssertions {
			maxAssertions = tf.AssertionCount
			maxEntity = tf.Name
		}
	}

	metrics.AssertionDensity = types.MetricSummary{
		Avg:       float64(totalAssertions) / float64(len(metrics.TestFunctions)),
		Max:       maxAssertions,
		MaxEntity: maxEntity,
	}
}
