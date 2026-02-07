package c1

import (
	"bytes"
	"fmt"
	"hash"
	"hash/fnv"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// dupSeq represents a duplicate code sequence.
type dupSeq struct {
	hash      uint64
	file      string
	startLine int
	endLine   int
}

// dupConfig holds language-specific configuration for duplication analysis.
type dupConfig struct {
	blockKinds     []string // AST node kinds that contain statement blocks
	skipKinds      []string // AST node kinds to skip during statement collection
	hashNode       func(hash.Hash64, *tree_sitter.Node, int) // Language-specific node hasher
}

// analyzeTSDuplication performs generic duplication analysis on Tree-sitter parsed files.
func analyzeTSDuplication(files []*parser.ParsedTreeSitterFile, cfg dupConfig) ([]types.DuplicateBlock, float64) {
	const minStatements = 3
	const minLines = 6

	var sequences []dupSeq
	totalLines := 0

	for _, f := range files {
		totalLines += bytes.Count(f.Content, []byte("\n")) + 1
		root := f.Tree.RootNode()
		collectDupSequences(root, f.RelPath, f.Content, minStatements, minLines, cfg, &sequences)
	}

	// Group by hash
	groups := make(map[uint64][]dupSeq)
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

				for _, s := range []dupSeq{a, b} {
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

// collectDupSequences walks the AST collecting hashed statement sequences from block nodes.
func collectDupSequences(node *tree_sitter.Node, file string, content []byte, minStmts, minLines int, cfg dupConfig, seqs *[]dupSeq) {
	if node == nil {
		return
	}

	kind := node.Kind()

	// Check if this is a block kind
	isBlock := false
	for _, bk := range cfg.blockKinds {
		if kind == bk {
			isBlock = true
			break
		}
	}

	if isBlock {
		var stmts []*tree_sitter.Node
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			ck := child.Kind()

			// Check if this kind should be skipped
			skip := false
			for _, sk := range cfg.skipKinds {
				if ck == sk {
					skip = true
					break
				}
			}
			if skip {
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

				h := hashStmtSequence(window, cfg.hashNode)
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
			collectDupSequences(child, file, content, minStmts, minLines, cfg, seqs)
		}
	}
}

// hashStmtSequence computes an FNV hash of a sequence of AST nodes.
func hashStmtSequence(stmts []*tree_sitter.Node, hashNode func(hash.Hash64, *tree_sitter.Node, int)) uint64 {
	h := fnv.New64a()
	for _, stmt := range stmts {
		hashNode(h, stmt, 0)
	}
	return h.Sum64()
}

// baseHashNodeStructure writes a common structural representation of an AST node to the hasher.
// Returns true if the node was fully handled, false if language-specific handling is needed.
func baseHashNodeStructure(h hash.Hash64, node *tree_sitter.Node, depth int) bool {
	if node == nil || depth > 5 {
		return true
	}

	kind := node.Kind()
	fmt.Fprint(h, kind)

	childCount := node.ChildCount()
	fmt.Fprintf(h, ":%d", childCount)

	// All languages skip "comment" and "identifier" children deeper recursion
	if kind == "comment" || kind == "identifier" {
		return true
	}

	return false
}

// analyzeTSFileSizes computes file size metrics (avg, max) for Tree-sitter parsed files.
func analyzeTSFileSizes(files []*parser.ParsedTreeSitterFile) types.MetricSummary {
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

	return types.MetricSummary{
		Avg:       float64(sum) / float64(len(sizes)),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}
}
