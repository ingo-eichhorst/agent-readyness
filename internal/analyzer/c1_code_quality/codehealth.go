package c1

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
	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for C1 metrics computation.
const (
	modulePathMinParts = 3
	toPercentC1        = 100.0

	// Duplication detection thresholds shared across Go, Python, and TypeScript analyzers.
	dupMinStatements = 3 // Minimum statement count to detect as duplicate block
	dupMinLines      = 6 // Minimum line span to qualify as substantial duplication
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

// c1MetricsResult is the internal result type stored in AnalysisResult.Metrics["c1"].
type c1MetricsResult = types.C1Metrics

// SetGoPackages stores Go-specific parsed packages for use during Analyze.
func (a *C1Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
	a.pkgs = pkgs
}

// c1Accumulator collects intermediate results across languages before final metric assembly.
type c1Accumulator struct {
	functions    []types.FunctionMetric
	duplicates   []types.DuplicateBlock
	totalDupRate float64
	dupRateCount int
	fileSizes    []types.MetricSummary
}

// Analyze runs all 6 C1 sub-analyses on the given packages and returns
// a combined AnalysisResult with Category "C1".
func (a *C1Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	metrics := &c1MetricsResult{
		AfferentCoupling: make(map[string]int),
		EfferentCoupling: make(map[string]int),
	}
	acc := &c1Accumulator{}

	if a.pkgs != nil {
		a.accumulateGo(acc, metrics)
	}

	for _, target := range targets {
		a.accumulateTarget(target, acc)
	}

	assembleC1Metrics(metrics, acc)

	return &types.AnalysisResult{
		Name:     "C1: Code Health",
		Category: "C1",
		Metrics:  map[string]types.CategoryMetrics{"c1": metrics},
	}, nil
}

// accumulateGo runs Go-specific C1 analysis and adds results to the accumulator.
func (a *C1Analyzer) accumulateGo(acc *c1Accumulator, metrics *c1MetricsResult) {
	goFunctions, goDuplicates, goDupRate, goFileSize, goCoupling := a.analyzeGoC1()
	acc.functions = append(acc.functions, goFunctions...)
	acc.duplicates = append(acc.duplicates, goDuplicates...)
	if goDupRate > 0 {
		acc.totalDupRate += goDupRate
		acc.dupRateCount++
	}
	acc.fileSizes = append(acc.fileSizes, goFileSize)
	for k, v := range goCoupling.afferent {
		metrics.AfferentCoupling[k] = v
	}
	for k, v := range goCoupling.efferent {
		metrics.EfferentCoupling[k] = v
	}
}

// accumulateTarget dispatches a single analysis target to the appropriate language handler.
func (a *C1Analyzer) accumulateTarget(target *types.AnalysisTarget, acc *c1Accumulator) {
	if a.tsParser == nil {
		return
	}

	switch target.Language {
	case types.LangPython:
		parsed, err := a.tsParser.ParseTargetFiles(target)
		if err != nil {
			return
		}
		defer parser.CloseAll(parsed)
		srcFiles := pyFilterSourceFiles(parsed)
		acc.functions = append(acc.functions, pyAnalyzeFunctions(srcFiles)...)
		acc.fileSizes = append(acc.fileSizes, pyAnalyzeFileSizes(srcFiles))
		dups, rate := pyAnalyzeDuplication(srcFiles)
		accumulateDuplication(acc, dups, rate)

	case types.LangTypeScript:
		parsed, err := a.tsParser.ParseTargetFiles(target)
		if err != nil {
			return
		}
		defer parser.CloseAll(parsed)
		srcFiles := tsFilterSourceFiles(parsed)
		acc.functions = append(acc.functions, tsAnalyzeFunctions(srcFiles)...)
		acc.fileSizes = append(acc.fileSizes, tsAnalyzeFileSizes(srcFiles))
		dups, rate := tsAnalyzeDuplication(srcFiles)
		accumulateDuplication(acc, dups, rate)
	}
}

// accumulateDuplication adds duplication results to the accumulator.
func accumulateDuplication(acc *c1Accumulator, dups []types.DuplicateBlock, rate float64) {
	acc.duplicates = append(acc.duplicates, dups...)
	if rate > 0 {
		acc.totalDupRate += rate
		acc.dupRateCount++
	}
}

// assembleC1Metrics builds the final C1 metrics from accumulated results.
func assembleC1Metrics(metrics *c1MetricsResult, acc *c1Accumulator) {
	metrics.Functions = acc.functions
	metrics.CyclomaticComplexity = computeComplexitySummary(acc.functions)
	metrics.FunctionLength = computeFunctionLengthSummary(acc.functions)
	metrics.DuplicatedBlocks = acc.duplicates
	metrics.FileSize = mergeFileSizes(acc.fileSizes)
	if acc.dupRateCount > 0 {
		metrics.DuplicationRate = acc.totalDupRate / float64(acc.dupRateCount)
	}
}

// mergeFileSizes picks the best file size summary across languages using max-based approach.
func mergeFileSizes(sizes []types.MetricSummary) types.MetricSummary {
	if len(sizes) == 0 {
		return types.MetricSummary{}
	}
	best := sizes[0]
	for _, fs := range sizes[1:] {
		if fs.Max > best.Max {
			best.Max = fs.Max
			best.MaxEntity = fs.MaxEntity
		}
	}
	return best
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
	graph := shared.BuildImportGraph(srcPkgs, modulePath)
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
// Computes cyclomatic complexity for all functions using gocyclo.
// Matches complexity results to function declarations by position (line number).
// Minimum complexity is 1 (function with no branches has complexity=1).
func analyzeFunctions(pkgs []*parser.ParsedPackage) []types.FunctionMetric {
	var allFunctions []types.FunctionMetric

	for _, pkg := range pkgs {
		complexityMap := buildComplexityMap(pkg)
		extractFunctionMetrics(pkg, complexityMap, &allFunctions)
	}

	return allFunctions
}

// posKey represents a file and line number for complexity lookup.
type posKey struct {
	file string
	line int
}

// buildComplexityMap computes complexity for all functions in a package.
func buildComplexityMap(pkg *parser.ParsedPackage) map[posKey]int {
	var stats gocyclo.Stats
	for _, f := range pkg.Syntax {
		stats = gocyclo.AnalyzeASTFile(f, pkg.Fset, stats)
	}

	complexityByPos := make(map[posKey]int)
	for _, s := range stats {
		complexityByPos[posKey{s.Pos.Filename, s.Pos.Line}] = s.Complexity
	}
	return complexityByPos
}

// extractFunctionMetrics walks the AST extracting function metrics.
func extractFunctionMetrics(pkg *parser.ParsedPackage, complexityMap map[posKey]int, allFunctions *[]types.FunctionMetric) {
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}

			metric := buildFunctionMetric(pkg, fn, complexityMap)
			*allFunctions = append(*allFunctions, metric)
			return true
		})
	}
}

// buildFunctionMetric creates a FunctionMetric from a function declaration.
func buildFunctionMetric(pkg *parser.ParsedPackage, fn *ast.FuncDecl, complexityMap map[posKey]int) types.FunctionMetric {
	pos := pkg.Fset.Position(fn.Pos())
	end := pkg.Fset.Position(fn.End())
	lineCount := end.Line - pos.Line + 1

	name := fn.Name.Name
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		// Method: prepend receiver type
		name = fmt.Sprintf("%s.%s", receiverTypeName(fn.Recv.List[0].Type), fn.Name.Name)
	}

	complexity := complexityMap[posKey{pos.Filename, pos.Line}]
	if complexity == 0 {
		complexity = 1 // minimum complexity
	}

	return types.FunctionMetric{
		Package:    pkg.PkgPath,
		Name:       name,
		File:       pos.Filename,
		Line:       pos.Line,
		Complexity: complexity,
		LineCount:  lineCount,
	}
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
		if len(parts) >= modulePathMinParts {
			return strings.Join(parts[:modulePathMinParts], "/")
		}
		return path
	}
	return ""
}

// analyzeDuplication detects duplicate code blocks using AST statement-sequence hashing.
//
// Algorithm approach:
// - Sliding window over statement sequences within each block
// - Structural hashing normalizes variable names to detect logic patterns
// - Groups sequences by hash to find matches across the codebase
//
// Thresholds:
// - dupMinStatements=3: Reduces false positives from trivial assignments/returns
// - dupMinLines=6: Focuses on substantial duplicated logic worth refactoring
//
// Returns the list of duplicate blocks and the duplication rate (% of lines duplicated).
// stmtSeq represents a hashed statement sequence with its source location.
type stmtSeq struct {
	hash      uint64
	file      string
	startLine int
	endLine   int
}

// analyzeDuplication detects duplicate code blocks using AST statement-sequence hashing.
func analyzeDuplication(pkgs []*parser.ParsedPackage) ([]types.DuplicateBlock, float64) {
	sequences, totalLines := collectStatementSequences(pkgs)
	groups := groupSequencesByHash(sequences)
	blocks, duplicatedLines := findDuplicatePairs(groups)
	rate := computeDuplicationRate(duplicatedLines, totalLines)
	return blocks, rate
}

// collectStatementSequences extracts all hashed statement windows from packages.
func collectStatementSequences(pkgs []*parser.ParsedPackage) ([]stmtSeq, int) {
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
				sequences = collectBlockWindows(pkg, block, sequences)
				return true
			})
		}
	}
	return sequences, totalLines
}

// collectBlockWindows generates sliding-window statement hashes for a single block.
func collectBlockWindows(pkg *parser.ParsedPackage, block *ast.BlockStmt, sequences []stmtSeq) []stmtSeq {
	for i := 0; i <= len(block.List)-dupMinStatements; i++ {
		for windowSize := dupMinStatements; windowSize <= len(block.List)-i; windowSize++ {
			stmts := block.List[i : i+windowSize]
			start := pkg.Fset.Position(stmts[0].Pos())
			end := pkg.Fset.Position(stmts[len(stmts)-1].End())
			if end.Line-start.Line+1 < dupMinLines {
				continue
			}
			sequences = append(sequences, stmtSeq{
				hash:      hashStatements(pkg.Fset, stmts),
				file:      start.Filename,
				startLine: start.Line,
				endLine:   end.Line,
			})
		}
	}
	return sequences
}

// groupSequencesByHash groups statement sequences by their hash value.
func groupSequencesByHash(sequences []stmtSeq) map[uint64][]stmtSeq {
	groups := make(map[uint64][]stmtSeq)
	for _, seq := range sequences {
		groups[seq.hash] = append(groups[seq.hash], seq)
	}
	return groups
}

// findDuplicatePairs finds non-overlapping duplicate pairs and tracks duplicated lines.
func findDuplicatePairs(groups map[uint64][]stmtSeq) ([]types.DuplicateBlock, map[string]map[int]bool) {
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
					FileA: a.file, StartA: a.startLine, EndA: a.endLine,
					FileB: b.file, StartB: b.startLine, EndB: b.endLine,
					LineCount: a.endLine - a.startLine + 1,
				})
				markDuplicatedLines(duplicatedLines, a)
				markDuplicatedLines(duplicatedLines, b)
			}
		}
	}
	return blocks, duplicatedLines
}

// markDuplicatedLines records all lines in a statement sequence as duplicated.
func markDuplicatedLines(duplicatedLines map[string]map[int]bool, s stmtSeq) {
	if duplicatedLines[s.file] == nil {
		duplicatedLines[s.file] = make(map[int]bool)
	}
	for l := s.startLine; l <= s.endLine; l++ {
		duplicatedLines[s.file][l] = true
	}
}

// computeDuplicationRate calculates the percentage of lines involved in duplication.
func computeDuplicationRate(duplicatedLines map[string]map[int]bool, totalLines int) float64 {
	if totalLines == 0 {
		return 0
	}
	dupLineCount := 0
	for _, lines := range duplicatedLines {
		dupLineCount += len(lines)
	}
	return float64(dupLineCount) / float64(totalLines) * toPercentC1
}

// hashStatements computes an FNV hash of a sequence of AST statements
// based on their normalized string representation.
//
// Normalization approach:
// - Statement types and operators are preserved (if, for, switch, assign, etc.)
// - Identifier names are abstracted to "id" via hashNode to match structurally similar code
// - Literal values are included to distinguish different constant usage patterns
//
// This allows detection of copy-pasted logic with renamed variables while
// avoiding false positives from semantically different code.
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

// hashNode recursively hashes an AST node using structural fingerprinting.
// Identifiers are normalized to "id" to ignore variable naming, preserving
// only the code structure. This enables detection of duplicated patterns
// regardless of local variable names.
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
