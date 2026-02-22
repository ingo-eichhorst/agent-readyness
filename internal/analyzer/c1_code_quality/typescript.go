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

// Constants for TypeScript C1 metrics computation.
const (
	p90PercentileTS      = 0.9
	toPercentC1TS        = 100.0
	maxHashNodeDepth     = 5
	maxHashNodeChildren  = 10
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
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = shared.NodeText(nameNode, content)
	}
	if className != "" {
		name = className + "." + name
	}
	return name
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
		idx := int(math.Ceil(float64(len(sorted))*p90PercentileTS)) - 1
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
//
// TypeScript-specific approach:
// - Uses Tree-sitter AST to identify statement sequences within blocks
// - Applies sliding window over statements in function/class bodies and statement blocks
// - Normalizes variable names via structural hashing to detect renamed copies
// - Thresholds: dupMinStatements=3, dupMinLines=6 (same as Go/Python for consistency)
//
// Returns duplicate blocks and duplication rate (% of lines duplicated).
//
// Why duplication matters for agents: When agents modify duplicated code, they
// must find and update ALL copies to maintain consistency. Missing even one copy
// creates subtle bugs where behavior diverges. High duplication (>10%) dramatically
// increases agent error rates because the "find all copies" step often fails.
// Agents lack the contextual memory to reliably track duplicates across files.
func tsAnalyzeDuplication(files []*parser.ParsedTreeSitterFile) ([]types.DuplicateBlock, float64) {
	sequences, totalLines := tsCollectAllDupSequences(files)
	groups := tsGroupSequencesByHash(sequences)
	blocks, duplicatedLines := tsFindDuplicateBlocks(groups)
	rate := tsCalculateDuplicationRate(duplicatedLines, totalLines)

	return blocks, rate
}

// tsCollectAllDupSequences collects duplicate sequences from all files.
func tsCollectAllDupSequences(files []*parser.ParsedTreeSitterFile) ([]tsDupSeq, int) {
	var sequences []tsDupSeq
	totalLines := 0

	for _, f := range files {
		totalLines += bytes.Count(f.Content, []byte("\n")) + 1
		root := f.Tree.RootNode()
		tsCollectDupSequences(root, f.RelPath, f.Content, dupMinStatements, dupMinLines, &sequences)
	}

	return sequences, totalLines
}

// tsGroupSequencesByHash groups sequences by their hash value.
func tsGroupSequencesByHash(sequences []tsDupSeq) map[uint64][]tsDupSeq {
	groups := make(map[uint64][]tsDupSeq)
	for _, seq := range sequences {
		groups[seq.hash] = append(groups[seq.hash], seq)
	}
	return groups
}

// tsFindDuplicateBlocks finds duplicate blocks from grouped sequences.
func tsFindDuplicateBlocks(groups map[uint64][]tsDupSeq) ([]types.DuplicateBlock, map[string]map[int]bool) {
	var blocks []types.DuplicateBlock
	duplicatedLines := make(map[string]map[int]bool)

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		tsProcessDuplicateGroup(group, &blocks, duplicatedLines)
	}

	return blocks, duplicatedLines
}

// tsProcessDuplicateGroup processes a group of duplicate sequences.
func tsProcessDuplicateGroup(group []tsDupSeq, blocks *[]types.DuplicateBlock, duplicatedLines map[string]map[int]bool) {
	for i := 0; i < len(group); i++ {
		for j := i + 1; j < len(group); j++ {
			a, b := group[i], group[j]

			if tsSequencesOverlap(a, b) {
				continue
			}

			*blocks = append(*blocks, types.DuplicateBlock{
				FileA:     a.file,
				StartA:    a.startLine,
				EndA:      a.endLine,
				FileB:     b.file,
				StartB:    b.startLine,
				EndB:      b.endLine,
				LineCount: a.endLine - a.startLine + 1,
			})

			tsMarkDuplicatedLines([]tsDupSeq{a, b}, duplicatedLines)
		}
	}
}

// tsSequencesOverlap checks if two sequences overlap in the same file.
func tsSequencesOverlap(a, b tsDupSeq) bool {
	return a.file == b.file && a.startLine < b.endLine && b.startLine < a.endLine
}

// tsMarkDuplicatedLines marks lines as duplicated in the tracking map.
func tsMarkDuplicatedLines(seqs []tsDupSeq, duplicatedLines map[string]map[int]bool) {
	for _, s := range seqs {
		if duplicatedLines[s.file] == nil {
			duplicatedLines[s.file] = make(map[int]bool)
		}
		for l := s.startLine; l <= s.endLine; l++ {
			duplicatedLines[s.file][l] = true
		}
	}
}

// tsCalculateDuplicationRate calculates the duplication rate from duplicated lines.
func tsCalculateDuplicationRate(duplicatedLines map[string]map[int]bool, totalLines int) float64 {
	dupLineCount := 0
	for _, lines := range duplicatedLines {
		dupLineCount += len(lines)
	}

	if totalLines > 0 {
		return float64(dupLineCount) / float64(totalLines) * toPercentC1TS
	}
	return 0
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
	if node == nil || depth > maxHashNodeDepth {
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

	for i := uint(0); i < childCount && i < maxHashNodeChildren; i++ {
		child := node.Child(i)
		if child != nil {
			tsHashNodeStructure(h, child, depth+1)
		}
	}
}

// Suppress unused import warnings.
var _ = os.PathSeparator
