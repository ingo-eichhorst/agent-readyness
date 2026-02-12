package c6

import (
	"bytes"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// pyDetectTests detects test functions in Python files and counts test vs source files.
// Returns test function metrics, test file count, and source file count.
func pyDetectTests(files []*parser.ParsedTreeSitterFile) ([]types.TestFunctionMetric, int, int) {
	var testFuncs []types.TestFunctionMetric
	testFileCount := 0
	srcFileCount := 0

	for _, f := range files {
		isTest := shared.IsTestFileByPath(f.RelPath)
		if isTest {
			testFileCount++
		} else {
			srcFileCount++
		}

		if !isTest {
			continue
		}

		// Walk test file for test functions (def test_*)
		root := f.Tree.RootNode()
		shared.WalkTree(root, func(node *tree_sitter.Node) {
			if node.Kind() != "function_definition" {
				return
			}

			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}

			name := shared.NodeText(nameNode, f.Content)
			if !strings.HasPrefix(name, "test_") {
				return
			}

			assertCount := pyCountAssertions(node, f.Content)

			testFuncs = append(testFuncs, types.TestFunctionMetric{
				Name:           name,
				File:           f.RelPath,
				Line:           int(nameNode.StartPosition().Row) + 1,
				AssertionCount: assertCount,
			})
		})
	}

	return testFuncs, testFileCount, srcFileCount
}

// pyCountAssertions counts assertion statements within a Python function body.
// Counts: assert statements, self.assert* calls, self.fail calls.
func pyCountAssertions(funcNode *tree_sitter.Node, content []byte) int {
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
		if kind == "function_definition" && n != funcNode {
			return
		}

		switch kind {
		case "assert_statement":
			count++
		case "call":
			// Check for self.assert* or self.fail calls
			fn := n.ChildByFieldName("function")
			if fn != nil && fn.Kind() == "attribute" {
				fnText := shared.NodeText(fn, content)
				if strings.HasPrefix(fnText, "self.assert") || fnText == "self.fail" {
					count++
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

// pyExternalDepModules are Python modules that indicate external/impure dependencies.
var pyExternalDepModules = map[string]bool{
	"requests":   true,
	"urllib":     true,
	"httpx":     true,
	"aiohttp":   true,
	"sqlalchemy": true,
	"psycopg2":  true,
	"pymongo":   true,
	"redis":     true,
	"boto3":     true,
	"socket":    true,
	"http":      true,
	"subprocess": true,
}

// pyAnalyzeIsolation checks test file imports for external/impure dependencies.
// Returns isolation score (0-100, where 100 = fully isolated).
func pyAnalyzeIsolation(files []*parser.ParsedTreeSitterFile, testFuncs []types.TestFunctionMetric) float64 {
	if len(testFuncs) == 0 {
		return 100 // No tests = vacuously isolated
	}

	// Check each test file for external imports
	testFileHasExtDep := make(map[string]bool)
	for _, f := range files {
		if !shared.IsTestFileByPath(f.RelPath) {
			continue
		}

		root := f.Tree.RootNode()
		hasExtDep := false

		shared.WalkTree(root, func(node *tree_sitter.Node) {
			kind := node.Kind()
			if kind != "import_statement" && kind != "import_from_statement" {
				return
			}

			// Extract module name
			var modName string
			switch kind {
			case "import_statement":
				for i := uint(0); i < node.ChildCount(); i++ {
					child := node.Child(i)
					if child != nil && (child.Kind() == "dotted_name" || child.Kind() == "aliased_import") {
						if child.Kind() == "aliased_import" {
							nameNode := child.ChildByFieldName("name")
							if nameNode != nil {
								modName = shared.NodeText(nameNode, f.Content)
							}
						} else {
							modName = shared.NodeText(child, f.Content)
						}
					}
				}
			case "import_from_statement":
				for i := uint(0); i < node.ChildCount(); i++ {
					child := node.Child(i)
					if child != nil && (child.Kind() == "dotted_name" || child.Kind() == "relative_import") {
						modName = shared.NodeText(child, f.Content)
						break
					}
				}
			}

			if modName == "" {
				return
			}

			// Check against known external deps
			topLevel := strings.Split(modName, ".")[0]
			if pyExternalDepModules[topLevel] {
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

	return float64(isolatedTests) / float64(totalTests) * toPercentC6
}

// pyCountLOC counts test LOC and source LOC for Python files.
func pyCountLOC(files []*parser.ParsedTreeSitterFile) (testLOC, srcLOC int) {
	for _, f := range files {
		lines := bytes.Count(f.Content, []byte("\n")) + 1
		if shared.IsTestFileByPath(f.RelPath) {
			testLOC += lines
		} else {
			srcLOC += lines
		}
	}
	return
}

// pyUpdateAssertionDensity recomputes assertion density from all test functions in metrics.
func pyUpdateAssertionDensity(metrics *types.C6Metrics) {
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
