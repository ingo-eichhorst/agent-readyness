package analyzer

import (
	"bytes"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"os"
	"sort"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// tsFilterSourceFiles filters to source-only TypeScript files (not test files).
func tsFilterSourceFiles(files []*parser.ParsedTreeSitterFile) []*parser.ParsedTreeSitterFile {
	var result []*parser.ParsedTreeSitterFile
	for _, f := range files {
		if tsIsTestFile(f.RelPath) {
			continue
		}
		result = append(result, f)
	}
	return result
}

// tsIsTestFile checks if a TypeScript file path indicates a test file.
func tsIsTestFile(path string) bool {
	lower := strings.ToLower(path)
	base := lower
	parts := strings.Split(lower, "/")
	if len(parts) > 0 {
		base = parts[len(parts)-1]
	}

	// Check __tests__ directory
	for _, p := range parts {
		if p == "__tests__" {
			return true
		}
	}

	return strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.tsx") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.js")
}

// tsAnalyzeFunctions extracts per-function complexity and line count from TypeScript files.
func tsAnalyzeFunctions(files []*parser.ParsedTreeSitterFile) []types.FunctionMetric {
	var results []types.FunctionMetric

	for _, f := range files {
		root := f.Tree.RootNode()
		tsWalkFunctions(root, f.Content, f.RelPath, "", &results)
	}

	return results
}

// tsWalkFunctions recursively walks the AST to find function declarations, arrow functions, and methods.
func tsWalkFunctions(node *tree_sitter.Node, content []byte, file string, className string, results *[]types.FunctionMetric) {
	if node == nil {
		return
	}

	kind := node.Kind()

	switch kind {
	case "class_declaration":
		nameNode := node.ChildByFieldName("name")
		clsName := ""
		if nameNode != nil {
			clsName = nodeText(nameNode, content)
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
		return

	case "function_declaration":
		nameNode := node.ChildByFieldName("name")
		name := ""
		if nameNode != nil {
			name = nodeText(nameNode, content)
		}
		if className != "" {
			name = className + "." + name
		}

		startRow := int(node.StartPosition().Row)
		endRow := int(node.EndPosition().Row)
		lineCount := endRow - startRow + 1
		complexity := tsComputeComplexity(node, content)

		*results = append(*results, types.FunctionMetric{
			Name:       name,
			File:       file,
			Line:       startRow + 1,
			Complexity: complexity,
			LineCount:  lineCount,
		})
		return

	case "method_definition":
		nameNode := node.ChildByFieldName("name")
		name := ""
		if nameNode != nil {
			name = nodeText(nameNode, content)
		}
		if className != "" {
			name = className + "." + name
		}

		startRow := int(node.StartPosition().Row)
		endRow := int(node.EndPosition().Row)
		lineCount := endRow - startRow + 1
		complexity := tsComputeComplexity(node, content)

		*results = append(*results, types.FunctionMetric{
			Name:       name,
			File:       file,
			Line:       startRow + 1,
			Complexity: complexity,
			LineCount:  lineCount,
		})
		return

	case "arrow_function":
		name := tsArrowFunctionName(node, content)
		startRow := int(node.StartPosition().Row)
		endRow := int(node.EndPosition().Row)
		lineCount := endRow - startRow + 1
		complexity := tsComputeComplexity(node, content)

		*results = append(*results, types.FunctionMetric{
			Name:       name,
			File:       file,
			Line:       startRow + 1,
			Complexity: complexity,
			LineCount:  lineCount,
		})
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

// tsArrowFunctionName tries to extract the name of an arrow function from its parent variable declarator.
func tsArrowFunctionName(node *tree_sitter.Node, content []byte) string {
	parent := node.Parent()
	if parent != nil && parent.Kind() == "variable_declarator" {
		nameNode := parent.ChildByFieldName("name")
		if nameNode != nil {
			return nodeText(nameNode, content)
		}
	}
	return "<anonymous>"
}

// tsComputeComplexity computes McCabe cyclomatic complexity for a TypeScript function.
// Base complexity is 1. Each branching construct adds 1.
// Nested function/arrow definitions are excluded.
func tsComputeComplexity(funcNode *tree_sitter.Node, content []byte) int {
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

		// Skip nested function/arrow definitions to avoid double-counting
		if kind == "function_declaration" || kind == "arrow_function" || kind == "function_expression" {
			return
		}

		switch kind {
		case "if_statement":
			complexity++
		case "for_statement", "for_in_statement":
			complexity++
		case "while_statement", "do_statement":
			complexity++
		case "switch_case":
			// Only count non-default cases
			// switch_case with a test expression is a case; without is default
			if n.ChildCount() > 0 {
				firstChild := n.Child(0)
				if firstChild != nil && nodeText(firstChild, content) != "default" {
					complexity++
				}
			}
		case "catch_clause":
			complexity++
		case "ternary_expression":
			complexity++
		case "binary_expression":
			// Count && || ?? operators
			opNode := n.ChildByFieldName("operator")
			if opNode != nil {
				op := nodeText(opNode, content)
				if op == "&&" || op == "||" || op == "??" {
					complexity++
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
	return complexity
}

// tsAnalyzeFileSizes computes file size metrics for TypeScript files.
func tsAnalyzeFileSizes(files []*parser.ParsedTreeSitterFile) types.MetricSummary {
	if len(files) == 0 {
		return types.MetricSummary{}
	}

	var sizes []int
	maxVal := 0
	maxEntity := ""

	for _, f := range files {
		lines := bytes.Count(f.Content, []byte("\n")) + 1
		sizes = append(sizes, lines)
		if lines > maxVal {
			maxVal = lines
			maxEntity = f.RelPath
		}
	}

	sum := 0
	for _, s := range sizes {
		sum += s
	}

	// Compute P90 (for future use)
	if len(sizes) > 0 {
		sorted := make([]int, len(sizes))
		copy(sorted, sizes)
		sort.Ints(sorted)
		idx := int(math.Ceil(float64(len(sorted))*0.9)) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		_ = sorted[idx]
	}

	return types.MetricSummary{
		Avg:       float64(sum) / float64(len(sizes)),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}
}

// tsDupSeq represents a hashed statement sequence for duplication detection.
type tsDupSeq struct {
	hash      uint64
	file      string
	startLine int
	endLine   int
}

// tsAnalyzeDuplication detects duplicate code blocks in TypeScript using statement-sequence hashing.
func tsAnalyzeDuplication(files []*parser.ParsedTreeSitterFile) ([]types.DuplicateBlock, float64) {
	const minStatements = 3
	const minLines = 6

	var sequences []tsDupSeq
	totalLines := 0

	for _, f := range files {
		totalLines += bytes.Count(f.Content, []byte("\n")) + 1
		root := f.Tree.RootNode()
		tsCollectDupSequences(root, f.RelPath, f.Content, minStatements, minLines, &sequences)
	}

	// Group by hash
	groups := make(map[uint64][]tsDupSeq)
	for _, seq := range sequences {
		groups[seq.hash] = append(groups[seq.hash], seq)
	}

	var blocks []types.DuplicateBlock
	duplicatedLines := make(map[string]map[int]bool)

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]

				if a.file == b.file && a.startLine < b.endLine && b.startLine < a.endLine {
					continue
				}

				blocks = append(blocks, types.DuplicateBlock{
					FileA:     a.file,
					StartA:    a.startLine,
					EndA:      a.endLine,
					FileB:     b.file,
					StartB:    b.startLine,
					EndB:      b.endLine,
					LineCount: a.endLine - a.startLine + 1,
				})

				for _, s := range []tsDupSeq{a, b} {
					if duplicatedLines[s.file] == nil {
						duplicatedLines[s.file] = make(map[int]bool)
					}
					for l := s.startLine; l <= s.endLine; l++ {
						duplicatedLines[s.file][l] = true
					}
				}
			}
		}
	}

	dupLineCount := 0
	for _, lines := range duplicatedLines {
		dupLineCount += len(lines)
	}

	var rate float64
	if totalLines > 0 {
		rate = float64(dupLineCount) / float64(totalLines) * 100
	}

	return blocks, rate
}

// tsCollectDupSequences walks the AST collecting hashed statement sequences from statement_block nodes.
func tsCollectDupSequences(node *tree_sitter.Node, file string, content []byte, minStmts, minLines int, seqs *[]tsDupSeq) {
	if node == nil {
		return
	}

	kind := node.Kind()

	if kind == "statement_block" || kind == "program" {
		var stmts []*tree_sitter.Node
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "comment" || ck == "" || ck == "{" || ck == "}" {
				continue
			}
			stmts = append(stmts, child)
		}

		for i := 0; i <= len(stmts)-minStmts; i++ {
			for windowSize := minStmts; windowSize <= len(stmts)-i; windowSize++ {
				window := stmts[i : i+windowSize]
				startLine := int(window[0].StartPosition().Row) + 1
				endLine := int(window[len(window)-1].EndPosition().Row) + 1
				lineSpan := endLine - startLine + 1

				if lineSpan < minLines {
					continue
				}

				h := tsHashStmtSequence(window)
				*seqs = append(*seqs, tsDupSeq{
					hash:      h,
					file:      file,
					startLine: startLine,
					endLine:   endLine,
				})
			}
		}
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil {
			tsCollectDupSequences(child, file, content, minStmts, minLines, seqs)
		}
	}
}

// tsHashStmtSequence computes an FNV hash of a sequence of TypeScript AST nodes.
func tsHashStmtSequence(stmts []*tree_sitter.Node) uint64 {
	h := fnv.New64a()
	for _, stmt := range stmts {
		tsHashNodeStructure(h, stmt, 0)
	}
	return h.Sum64()
}

// tsHashNodeStructure writes a structural representation of a TypeScript AST node to the hasher.
func tsHashNodeStructure(h hash.Hash64, node *tree_sitter.Node, depth int) {
	if node == nil || depth > 5 {
		return
	}

	kind := node.Kind()
	fmt.Fprint(h, kind)

	childCount := node.ChildCount()
	fmt.Fprintf(h, ":%d", childCount)

	switch kind {
	case "call_expression":
		fn := node.ChildByFieldName("function")
		if fn != nil {
			fmt.Fprint(h, fn.Kind())
		}
	case "assignment_expression":
		fmt.Fprint(h, "=")
	case "return_statement":
		fmt.Fprint(h, "ret")
	}

	for i := uint(0); i < childCount && i < 10; i++ {
		child := node.Child(i)
		if child != nil {
			tsHashNodeStructure(h, child, depth+1)
		}
	}
}

// Suppress unused import warnings.
var _ = os.PathSeparator
