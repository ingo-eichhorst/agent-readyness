package c1

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

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for Python C1 metrics computation.
const (
	p90PercentilePy     = 0.9
	toPercentC1Py       = 100.0
	maxHashNodeDepthPy  = 5
	maxHashNodeChildPy  = 10
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

// pyAnalyzeFileSizes computes file size metrics for Python files.
func pyAnalyzeFileSizes(files []*parser.ParsedTreeSitterFile) types.MetricSummary {
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
		idx := int(math.Ceil(float64(len(sorted))*p90PercentilePy)) - 1
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

// pyDupSeq represents a hashed statement sequence for duplication detection.
type pyDupSeq struct {
	hash      uint64
	file      string
	startLine int
	endLine   int
}

// pyAnalyzeDuplication detects duplicate code blocks in Python using statement-sequence hashing.
//
// Python-specific approach:
// - Uses Tree-sitter AST to identify statement sequences within blocks
// - Applies sliding window over statements in function/class bodies
// - Normalizes variable names via structural hashing to detect renamed copies
// - Thresholds: dupMinStatements=3, dupMinLines=6 (same as Go analysis for consistency)
//
// Returns duplicate blocks and duplication rate (% of lines duplicated).
func pyAnalyzeDuplication(files []*parser.ParsedTreeSitterFile) ([]types.DuplicateBlock, float64) {
	var sequences []pyDupSeq
	totalLines := 0

	for _, f := range files {
		totalLines += bytes.Count(f.Content, []byte("\n")) + 1
		root := f.Tree.RootNode()
		pyCollectDupSequences(root, f.RelPath, f.Content, dupMinStatements, dupMinLines, &sequences)
	}

	// Group by hash
	groups := make(map[uint64][]pyDupSeq)
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

				for _, s := range []pyDupSeq{a, b} {
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
		rate = float64(dupLineCount) / float64(totalLines) * toPercentC1Py
	}

	return blocks, rate
}

// pyCollectDupSequences walks the AST collecting hashed statement sequences from block nodes.
//
// Sliding window approach:
// - Identifies block and module nodes containing statement sequences
// - Filters out comments, newlines, and empty nodes
// - Applies sliding window of varying sizes to capture duplicates of different lengths
// - Only records sequences meeting minimum statement count and line span thresholds
func pyCollectDupSequences(node *tree_sitter.Node, file string, content []byte, minStmts, minLines int, seqs *[]pyDupSeq) {
	if node == nil {
		return
	}

	kind := node.Kind()

	if kind == "block" || kind == "module" {
		var stmts []*tree_sitter.Node
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			ck := child.Kind()
			if ck == "comment" || ck == "newline" || ck == "" {
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

				h := pyHashStmtSequence(window)
				*seqs = append(*seqs, pyDupSeq{
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
			pyCollectDupSequences(child, file, content, minStmts, minLines, seqs)
		}
	}
}

// pyHashStmtSequence computes an FNV hash of a sequence of Python AST nodes.
func pyHashStmtSequence(stmts []*tree_sitter.Node) uint64 {
	h := fnv.New64a()
	for _, stmt := range stmts {
		pyHashNodeStructure(h, stmt, 0)
	}
	return h.Sum64()
}

// pyHashNodeStructure writes a structural representation of a Python AST node to the hasher.
func pyHashNodeStructure(h hash.Hash64, node *tree_sitter.Node, depth int) {
	if node == nil || depth > maxHashNodeDepthPy {
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

	for i := uint(0); i < childCount && i < maxHashNodeChildPy; i++ {
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
