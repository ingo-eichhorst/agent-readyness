package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"hash"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/fzipp/gocyclo"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C1Analyzer implements the pipeline.Analyzer interface for C1: Code Health metrics.
// It also implements GoAwareAnalyzer for Go-specific analysis via SetGoPackages.
type C1Analyzer struct {
	pkgs     []*parser.ParsedPackage
	tsParser *parser.TreeSitterParser
}

// NewC1Analyzer creates a C1Analyzer with Tree-sitter parser for multi-language analysis.
func NewC1Analyzer(tsParser *parser.TreeSitterParser) *C1Analyzer {
	return &C1Analyzer{tsParser: tsParser}
}

// Name returns the analyzer display name.
func (a *C1Analyzer) Name() string {
	return "C1: Code Health"
}

// C1MetricsResult is the internal result type stored in AnalysisResult.Metrics["c1"].
type C1MetricsResult = types.C1Metrics

// SetGoPackages stores Go-specific parsed packages for use during Analyze.
func (a *C1Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
	a.pkgs = pkgs
}

// Analyze runs all 6 C1 sub-analyses on the given packages and returns
// a combined AnalysisResult with Category "C1".
func (a *C1Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	metrics := &C1MetricsResult{
		AfferentCoupling: make(map[string]int),
		EfferentCoupling: make(map[string]int),
	}

	var allFunctions []types.FunctionMetric
	var allDuplicates []types.DuplicateBlock
	var totalDupRate float64
	dupRateCount := 0
	var allFileSizes []types.MetricSummary

	// Go analysis (existing logic, via SetGoPackages)
	if a.pkgs != nil {
		goFunctions, goDuplicates, goDupRate, goFileSize, goCoupling := a.analyzeGoC1()
		allFunctions = append(allFunctions, goFunctions...)
		allDuplicates = append(allDuplicates, goDuplicates...)
		if goDupRate > 0 {
			totalDupRate += goDupRate
			dupRateCount++
		}
		allFileSizes = append(allFileSizes, goFileSize)
		// Apply Go coupling directly
		for k, v := range goCoupling.afferent {
			metrics.AfferentCoupling[k] = v
		}
		for k, v := range goCoupling.efferent {
			metrics.EfferentCoupling[k] = v
		}
	}

	// Python/TypeScript analysis via targets
	for _, target := range targets {
		switch target.Language {
		case types.LangPython:
			if a.tsParser == nil {
				continue
			}
			parsed, err := a.tsParser.ParseTargetFiles(target)
			if err != nil {
				continue
			}
			defer parser.CloseAll(parsed)

			srcFiles := pyFilterSourceFiles(parsed)
			pyFunctions := pyAnalyzeFunctions(srcFiles)
			allFunctions = append(allFunctions, pyFunctions...)

			pyFileSize := pyAnalyzeFileSizes(srcFiles)
			allFileSizes = append(allFileSizes, pyFileSize)

			pyDups, pyRate := pyAnalyzeDuplication(srcFiles)
			allDuplicates = append(allDuplicates, pyDups...)
			if pyRate > 0 {
				totalDupRate += pyRate
				dupRateCount++
			}

		case types.LangTypeScript:
			if a.tsParser == nil {
				continue
			}
			parsed, err := a.tsParser.ParseTargetFiles(target)
			if err != nil {
				continue
			}
			defer parser.CloseAll(parsed)

			srcFiles := tsFilterSourceFiles(parsed)
			tsFunctions := tsAnalyzeFunctions(srcFiles)
			allFunctions = append(allFunctions, tsFunctions...)

			tsFileSize := tsAnalyzeFileSizes(srcFiles)
			allFileSizes = append(allFileSizes, tsFileSize)

			tsDups, tsRate := tsAnalyzeDuplication(srcFiles)
			allDuplicates = append(allDuplicates, tsDups...)
			if tsRate > 0 {
				totalDupRate += tsRate
				dupRateCount++
			}
		}
	}

	// Build combined metrics
	metrics.Functions = allFunctions
	metrics.CyclomaticComplexity = computeComplexitySummary(allFunctions)
	metrics.FunctionLength = computeFunctionLengthSummary(allFunctions)
	metrics.DuplicatedBlocks = allDuplicates

	// Merge file sizes: pick the best summary across languages
	if len(allFileSizes) > 0 {
		bestFileSize := allFileSizes[0]
		for _, fs := range allFileSizes[1:] {
			if fs.Max > bestFileSize.Max {
				bestFileSize.Max = fs.Max
				bestFileSize.MaxEntity = fs.MaxEntity
			}
			// Re-average would need total count; use max-based approach
		}
		metrics.FileSize = bestFileSize
	}

	// Average duplication rate
	if dupRateCount > 0 {
		metrics.DuplicationRate = totalDupRate / float64(dupRateCount)
	}

	return &types.AnalysisResult{
		Name:     "C1: Code Health",
		Category: "C1",
		Metrics:  map[string]interface{}{"c1": metrics},
	}, nil
}

// goCouplingResult holds afferent and efferent coupling maps from Go analysis.
type goCouplingResult struct {
	afferent map[string]int
	efferent map[string]int
}

// analyzeGoC1 runs Go-specific C1 analysis and returns its components.
func (a *C1Analyzer) analyzeGoC1() ([]types.FunctionMetric, []types.DuplicateBlock, float64, types.MetricSummary, goCouplingResult) {
	pkgs := a.pkgs
	var srcPkgs []*parser.ParsedPackage
	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			continue
		}
		srcPkgs = append(srcPkgs, pkg)
	}

	functions := analyzeFunctions(srcPkgs)
	fileSize := analyzeFileSizes(srcPkgs)
	duplicates, dupRate := analyzeDuplication(srcPkgs)

	// Coupling
	modulePath := detectModulePath(srcPkgs)
	graph := BuildImportGraph(srcPkgs, modulePath)
	coupling := goCouplingResult{
		afferent: make(map[string]int),
		efferent: make(map[string]int),
	}
	for _, pkg := range srcPkgs {
		coupling.afferent[pkg.PkgPath] = len(graph.Reverse[pkg.PkgPath])
		coupling.efferent[pkg.PkgPath] = len(graph.Forward[pkg.PkgPath])
	}

	return functions, duplicates, dupRate, fileSize, coupling
}

// analyzeFunctions extracts per-function complexity and line count from all source packages.
func analyzeFunctions(pkgs []*parser.ParsedPackage) []types.FunctionMetric {
	var allFunctions []types.FunctionMetric

	for _, pkg := range pkgs {
		// Use gocyclo for complexity
		var stats gocyclo.Stats
		for _, f := range pkg.Syntax {
			stats = gocyclo.AnalyzeASTFile(f, pkg.Fset, stats)
		}

		// Build complexity map by position for matching
		type posKey struct {
			file string
			line int
		}
		complexityByPos := make(map[posKey]int)
		for _, s := range stats {
			complexityByPos[posKey{s.Pos.Filename, s.Pos.Line}] = s.Complexity
		}

		// Walk AST for function declarations to get line counts
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					return true
				}

				pos := pkg.Fset.Position(fn.Pos())
				end := pkg.Fset.Position(fn.End())
				lineCount := end.Line - pos.Line + 1

				name := fn.Name.Name
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					// Method: prepend receiver type
					name = fmt.Sprintf("%s.%s", receiverTypeName(fn.Recv.List[0].Type), fn.Name.Name)
				}

				complexity := complexityByPos[posKey{pos.Filename, pos.Line}]
				if complexity == 0 {
					complexity = 1 // minimum complexity
				}

				allFunctions = append(allFunctions, types.FunctionMetric{
					Package:    pkg.PkgPath,
					Name:       name,
					File:       pos.Filename,
					Line:       pos.Line,
					Complexity: complexity,
					LineCount:  lineCount,
				})

				return true
			})
		}
	}

	return allFunctions
}

// receiverTypeName extracts the type name from a receiver expression.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	default:
		return "?"
	}
}

// computeComplexitySummary calculates avg and max cyclomatic complexity.
func computeComplexitySummary(functions []types.FunctionMetric) types.MetricSummary {
	if len(functions) == 0 {
		return types.MetricSummary{}
	}

	sum := 0
	maxVal := 0
	maxEntity := ""

	for _, f := range functions {
		sum += f.Complexity
		if f.Complexity > maxVal {
			maxVal = f.Complexity
			maxEntity = f.Name
		}
	}

	return types.MetricSummary{
		Avg:       float64(sum) / float64(len(functions)),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}
}

// computeFunctionLengthSummary calculates avg and max function length.
func computeFunctionLengthSummary(functions []types.FunctionMetric) types.MetricSummary {
	if len(functions) == 0 {
		return types.MetricSummary{}
	}

	sum := 0
	maxVal := 0
	maxEntity := ""

	for _, f := range functions {
		sum += f.LineCount
		if f.LineCount > maxVal {
			maxVal = f.LineCount
			maxEntity = f.Name
		}
	}

	return types.MetricSummary{
		Avg:       float64(sum) / float64(len(functions)),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}
}

// analyzeFileSizes measures lines per file across all source packages.
func analyzeFileSizes(pkgs []*parser.ParsedPackage) types.MetricSummary {
	var sum int
	var count int
	maxVal := 0
	maxEntity := ""

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			lines := pkg.Fset.Position(f.End()).Line
			sum += lines
			count++
			if lines > maxVal {
				maxVal = lines
				maxEntity = pkg.Fset.Position(f.Pos()).Filename
			}
		}
	}

	if count == 0 {
		return types.MetricSummary{}
	}

	return types.MetricSummary{
		Avg:       float64(sum) / float64(count),
		Max:       maxVal,
		MaxEntity: maxEntity,
	}
}

// detectModulePath extracts the module path from go.mod in the package directory,
// or infers it from the first package's import path.
func detectModulePath(pkgs []*parser.ParsedPackage) string {
	if len(pkgs) == 0 {
		return ""
	}

	// Try to find go.mod by walking up from the first package's file
	if len(pkgs[0].GoFiles) > 0 {
		dir := filepath.Dir(pkgs[0].GoFiles[0])
		for {
			modFile := filepath.Join(dir, "go.mod")
			data, err := os.ReadFile(modFile)
			if err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "module ") {
						return strings.TrimSpace(strings.TrimPrefix(line, "module"))
					}
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Fallback: use common prefix of package paths
	if len(pkgs) > 0 {
		path := pkgs[0].PkgPath
		// Use everything up to the first package component
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/")
		}
		return path
	}
	return ""
}

// analyzeDuplication detects duplicate code blocks using AST statement-sequence hashing.
// Returns the list of duplicate blocks and the duplication rate (0-100).
func analyzeDuplication(pkgs []*parser.ParsedPackage) ([]types.DuplicateBlock, float64) {
	const minStatements = 3
	const minLines = 6

	type stmtSeq struct {
		hash      uint64
		file      string
		startLine int
		endLine   int
	}

	var sequences []stmtSeq
	totalLines := 0

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			totalLines += pkg.Fset.Position(f.End()).Line

			ast.Inspect(f, func(n ast.Node) bool {
				block, ok := n.(*ast.BlockStmt)
				if !ok {
					return true
				}

				// Sliding window over statements
				for i := 0; i <= len(block.List)-minStatements; i++ {
					for windowSize := minStatements; windowSize <= len(block.List)-i; windowSize++ {
						stmts := block.List[i : i+windowSize]
						start := pkg.Fset.Position(stmts[0].Pos())
						end := pkg.Fset.Position(stmts[len(stmts)-1].End())
						lineSpan := end.Line - start.Line + 1

						if lineSpan < minLines {
							continue
						}

						h := hashStatements(pkg.Fset, stmts)
						sequences = append(sequences, stmtSeq{
							hash:      h,
							file:      start.Filename,
							startLine: start.Line,
							endLine:   end.Line,
						})
					}
				}

				return true
			})
		}
	}

	// Group by hash and find duplicates
	groups := make(map[uint64][]stmtSeq)
	for _, seq := range sequences {
		groups[seq.hash] = append(groups[seq.hash], seq)
	}

	var blocks []types.DuplicateBlock
	duplicatedLines := make(map[string]map[int]bool) // file -> set of duplicated line numbers

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}

		// Report pairs, but only the largest non-overlapping match per pair
		// For simplicity, report all pairs from the group
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				a, b := group[i], group[j]

				// Skip self-overlapping matches (same file, overlapping lines)
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

				// Track duplicated lines
				for _, s := range []stmtSeq{a, b} {
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

	// Calculate duplication rate
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

// hashStatements computes an FNV hash of a sequence of AST statements
// based on their normalized string representation.
func hashStatements(fset *token.FileSet, stmts []ast.Stmt) uint64 {
	h := fnv.New64a()
	for _, stmt := range stmts {
		// Hash the node type and structure, ignoring identifiers' specific names
		// but preserving structure. We use a simplified approach: hash the
		// statement type and relative position offsets.
		hashNode(h, fset, stmt)
	}
	return h.Sum64()
}

// hashNode writes a structural representation of an AST node to the hasher.
func hashNode(h hash.Hash64, fset *token.FileSet, node ast.Node) {
	if node == nil {
		h.Write([]byte("nil"))
		return
	}

	switch n := node.(type) {
	case *ast.AssignStmt:
		fmt.Fprintf(h, "assign:%d:%d", len(n.Lhs), n.Tok)
		for _, expr := range n.Rhs {
			hashExpr(h, expr)
		}
	case *ast.ExprStmt:
		h.Write([]byte("expr:"))
		hashExpr(h, n.X)
	case *ast.ReturnStmt:
		fmt.Fprintf(h, "return:%d", len(n.Results))
	case *ast.IfStmt:
		h.Write([]byte("if"))
	case *ast.ForStmt:
		h.Write([]byte("for"))
	case *ast.RangeStmt:
		h.Write([]byte("range"))
	case *ast.SwitchStmt:
		h.Write([]byte("switch"))
	case *ast.DeclStmt:
		h.Write([]byte("decl"))
		if gd, ok := n.Decl.(*ast.GenDecl); ok {
			fmt.Fprintf(h, ":%d", gd.Tok)
		}
	default:
		fmt.Fprintf(h, "other:%T", n)
	}
}

// hashExpr writes a structural representation of an expression to the hasher.
func hashExpr(h hash.Hash64, expr ast.Expr) {
	if expr == nil {
		h.Write([]byte("nil"))
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpr:
		h.Write([]byte("call:"))
		hashExpr(h, e.Fun)
		fmt.Fprintf(h, ":%d", len(e.Args))
		for _, arg := range e.Args {
			hashExpr(h, arg)
		}
	case *ast.SelectorExpr:
		h.Write([]byte("sel:"))
		hashExpr(h, e.X)
		h.Write([]byte("." + e.Sel.Name))
	case *ast.Ident:
		// Hash identifier usage pattern but not the specific name
		// to detect structurally similar code with different variable names
		h.Write([]byte("id"))
	case *ast.BasicLit:
		fmt.Fprintf(h, "lit:%s:%s", e.Kind, e.Value)
	case *ast.BinaryExpr:
		fmt.Fprintf(h, "bin:%s:", e.Op)
		hashExpr(h, e.X)
		hashExpr(h, e.Y)
	case *ast.UnaryExpr:
		fmt.Fprintf(h, "unary:%s:", e.Op)
		hashExpr(h, e.X)
	default:
		fmt.Fprintf(h, "expr:%T", e)
	}
}
