package c1

// Package-level duplication detection framework shared by Python and TypeScript analyzers.
// Language-specific behavior is injected via langDupConfig.

import (
	"bytes"
	"fmt"
	"hash"
	"hash/fnv"
	"sort"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// extractClassedFuncName extracts a function name node's text, prefixed with
// className if the function belongs to a class (e.g. "MyClass.doThing").
// Shared by Python and TypeScript function walkers.
func extractClassedFuncName(node *tree_sitter.Node, content []byte, className string) string {
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

const (
	dupHashMaxDepth    = 5
	dupHashMaxChildren = 10
	toPercent          = 100.0
)

// dupSeq represents a hashed statement sequence for duplication detection.
type dupSeq struct {
	hash      uint64
	file      string
	startLine int
	endLine   int
}

// langDupConfig holds language-specific configuration for duplication detection.
type langDupConfig struct {
	// blockKinds are AST node kinds that contain statement sequences (e.g. "block", "statement_block").
	blockKinds []string
	// skipKinds are child node kinds to skip when collecting statements (e.g. "comment", "newline").
	skipKinds []string
	// callKind is the AST node kind for function calls (e.g. "call" for Python, "call_expression" for TS).
	callKind string
	// assignKind is the AST node kind for assignments.
	assignKind string
}

// treeAnalyzeDuplication detects duplicate code blocks using statement-sequence hashing.
// Language-specific behavior is controlled by cfg.
func treeAnalyzeDuplication(files []*parser.ParsedTreeSitterFile, cfg langDupConfig) ([]types.DuplicateBlock, float64) {
	sequences, totalLines := treeCollectAllDupSequences(files, cfg)
	groups := treeGroupSequencesByHash(sequences)
	blocks, duplicatedLines := treeFindDuplicateBlocks(groups)
	rate := treeCalculateDuplicationRate(duplicatedLines, totalLines)
	return blocks, rate
}

// collectAllDupSequences collects all hashed sequences from all files.
func treeCollectAllDupSequences(files []*parser.ParsedTreeSitterFile, cfg langDupConfig) ([]dupSeq, int) {
	var sequences []dupSeq
	totalLines := 0
	for _, f := range files {
		totalLines += bytes.Count(f.Content, []byte("\n")) + 1
		root := f.Tree.RootNode()
		treeCollectDupSequences(root, f.RelPath, f.Content, dupMinStatements, dupMinLines, cfg, &sequences)
	}
	return sequences, totalLines
}

// groupSequencesByHash groups sequences by their hash value.
func treeGroupSequencesByHash(sequences []dupSeq) map[uint64][]dupSeq {
	groups := make(map[uint64][]dupSeq)
	for _, seq := range sequences {
		groups[seq.hash] = append(groups[seq.hash], seq)
	}
	return groups
}

// findDuplicateBlocks finds duplicate blocks from grouped sequences.
func treeFindDuplicateBlocks(groups map[uint64][]dupSeq) ([]types.DuplicateBlock, map[string]map[int]bool) {
	var blocks []types.DuplicateBlock
	duplicatedLines := make(map[string]map[int]bool)
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		treeProcessDuplicateGroup(group, &blocks, duplicatedLines)
	}
	return blocks, duplicatedLines
}

// processDuplicateGroup processes a group of duplicate sequences.
func treeProcessDuplicateGroup(group []dupSeq, blocks *[]types.DuplicateBlock, duplicatedLines map[string]map[int]bool) {
	for i := 0; i < len(group); i++ {
		for j := i + 1; j < len(group); j++ {
			a, b := group[i], group[j]
			if treeSequencesOverlap(a, b) {
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
			treeMarkDuplicatedLines([]dupSeq{a, b}, duplicatedLines)
		}
	}
}

// sequencesOverlap checks if two sequences overlap in the same file.
func treeSequencesOverlap(a, b dupSeq) bool {
	return a.file == b.file && a.startLine < b.endLine && b.startLine < a.endLine
}

// markDuplicatedLines marks lines as duplicated in the tracking map.
func treeMarkDuplicatedLines(seqs []dupSeq, duplicatedLines map[string]map[int]bool) {
	for _, s := range seqs {
		if duplicatedLines[s.file] == nil {
			duplicatedLines[s.file] = make(map[int]bool)
		}
		for l := s.startLine; l <= s.endLine; l++ {
			duplicatedLines[s.file][l] = true
		}
	}
}

// calculateDuplicationRate calculates the duplication rate as a percentage.
func treeCalculateDuplicationRate(duplicatedLines map[string]map[int]bool, totalLines int) float64 {
	dupLineCount := 0
	for _, lines := range duplicatedLines {
		dupLineCount += len(lines)
	}
	if totalLines > 0 {
		return float64(dupLineCount) / float64(totalLines) * toPercent
	}
	return 0
}

// collectDupSequences walks the AST collecting hashed statement sequences from block nodes.
func treeCollectDupSequences(node *tree_sitter.Node, file string, content []byte, minStmts, minLines int, cfg langDupConfig, seqs *[]dupSeq) {
	if node == nil {
		return
	}

	kind := node.Kind()
	if treeKindInList(kind, cfg.blockKinds) {
		var stmts []*tree_sitter.Node
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			if treeKindInList(child.Kind(), cfg.skipKinds) {
				continue
			}
			stmts = append(stmts, child)
		}

		for i := 0; i <= len(stmts)-minStmts; i++ {
			maxWin := len(stmts) - i
			if maxWin > dupMaxWindowSize {
				maxWin = dupMaxWindowSize
			}
			for windowSize := minStmts; windowSize <= maxWin; windowSize++ {
				window := stmts[i : i+windowSize]
				startLine := int(window[0].StartPosition().Row) + 1
				endLine := int(window[len(window)-1].EndPosition().Row) + 1
				if endLine-startLine+1 < minLines {
					continue
				}
				h := treeHashStmtSequence(window, cfg)
				*seqs = append(*seqs, dupSeq{
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
			treeCollectDupSequences(child, file, content, minStmts, minLines, cfg, seqs)
		}
	}
}

// treeKindInList checks if kind is present in the given list of node kinds.
// Used for both block-kind and skip-kind lookups.
func treeKindInList(kind string, kinds []string) bool {
	for _, k := range kinds {
		if kind == k {
			return true
		}
	}
	return false
}

// hashStmtSequence computes an FNV hash of a sequence of AST nodes.
func treeHashStmtSequence(stmts []*tree_sitter.Node, cfg langDupConfig) uint64 {
	h := fnv.New64a()
	for _, stmt := range stmts {
		treeHashNodeStructure(h, stmt, 0, cfg)
	}
	return h.Sum64()
}

// hashNodeStructure writes a structural representation of an AST node to the hasher.
func treeHashNodeStructure(h hash.Hash64, node *tree_sitter.Node, depth int, cfg langDupConfig) {
	if node == nil || depth > dupHashMaxDepth {
		return
	}

	kind := node.Kind()
	fmt.Fprint(h, kind)

	childCount := node.ChildCount()
	fmt.Fprintf(h, ":%d", childCount)

	switch kind {
	case cfg.callKind:
		fn := node.ChildByFieldName("function")
		if fn != nil {
			fmt.Fprint(h, fn.Kind())
		}
	case cfg.assignKind:
		fmt.Fprint(h, "=")
	case "return_statement":
		fmt.Fprint(h, "ret")
	}

	for i := uint(0); i < childCount && i < dupHashMaxChildren; i++ {
		child := node.Child(i)
		if child != nil {
			treeHashNodeStructure(h, child, depth+1, cfg)
		}
	}
}

// treeAnalyzeFileSizes computes file size metrics from a list of tree-sitter parsed files.
// Returns the metric summary and the individual file LOC counts.
func treeAnalyzeFileSizes(files []*parser.ParsedTreeSitterFile) (types.MetricSummary, []int) {
	if len(files) == 0 {
		return types.MetricSummary{}, nil
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

	// Compute P90 percentile (tracked but not yet surfaced in output).
	if len(sizes) > 0 {
		sorted := make([]int, len(sizes))
		copy(sorted, sizes)
		sort.Ints(sorted)
		idx := int(float64(len(sorted)) * 0.9)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		_ = sorted[idx]
	}

	return types.MetricSummary{
		Avg:       float64(sum) / float64(len(sizes)),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}, sizes
}

