package c3

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	arstypes "github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for C3 metrics computation.
const (
	modulePathMinPartsC3 = 3
)

// C3Analyzer implements the pipeline.Analyzer interface for C3: Architectural Navigability.
// It also implements GoAwareAnalyzer for Go-specific analysis via SetGoPackages.
type C3Analyzer struct {
	pkgs     []*parser.ParsedPackage
	tsParser *parser.TreeSitterParser
}

// NewC3Analyzer creates a C3Analyzer with Tree-sitter parser for multi-language analysis.
func NewC3Analyzer(tsParser *parser.TreeSitterParser) *C3Analyzer {
	return &C3Analyzer{tsParser: tsParser}
}

// Name returns the analyzer display name.
func (a *C3Analyzer) Name() string {
	return "C3: Architecture"
}

// SetGoPackages stores Go-specific parsed packages for use during Analyze.
func (a *C3Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
	a.pkgs = pkgs
}

// Analyze runs all 5 C3 sub-analyses and returns a combined AnalysisResult.
func (a *C3Analyzer) Analyze(targets []*arstypes.AnalysisTarget) (*arstypes.AnalysisResult, error) {
	metrics := &arstypes.C3Metrics{}

	if a.pkgs != nil {
		metrics = a.analyzeGoC3()
	}

	for _, target := range targets {
		a.analyzeTargetC3(target, metrics)
	}

	return &arstypes.AnalysisResult{
		Name:     "C3: Architecture",
		Category: "C3",
		Metrics:  map[string]arstypes.CategoryMetrics{"c3": metrics},
	}, nil
}

// analyzeTargetC3 runs C3 analysis for a single Python or TypeScript target.
func (a *C3Analyzer) analyzeTargetC3(target *arstypes.AnalysisTarget, metrics *arstypes.C3Metrics) {
	if a.tsParser == nil {
		return
	}

	parsed, err := a.tsParser.ParseTargetFiles(target)
	if err != nil {
		return
	}
	defer parser.CloseAll(parsed)

	var graph *shared.ImportGraph
	var dead []arstypes.DeadExport
	var maxDepth int
	var avgDepth float64
	var hasSrcFiles bool

	switch target.Language {
	case arstypes.LangPython:
		hasSrcFiles = len(pyFilterSourceFiles(parsed)) > 0
		graph = pyBuildImportGraph(parsed)
		dead = pyDetectDeadCode(parsed)
		maxDepth, avgDepth = pyAnalyzeDirectoryDepth(parsed, target.RootDir)
	case arstypes.LangTypeScript:
		hasSrcFiles = len(tsFilterSourceFiles(parsed)) > 0
		graph = tsBuildImportGraph(parsed)
		dead = tsDetectDeadCode(parsed)
		maxDepth, avgDepth = tsAnalyzeDirectoryDepth(parsed, target.RootDir)
	default:
		return
	}

	mergeDepthMetrics(metrics, maxDepth, avgDepth)
	metrics.CircularDeps = append(metrics.CircularDeps, detectCircularDeps(graph)...)
	if hasSrcFiles {
		mergeFanoutMetrics(metrics, graph)
	}
	metrics.DeadExports = append(metrics.DeadExports, dead...)
}

// mergeDepthMetrics updates metrics with deeper directory depth values.
func mergeDepthMetrics(metrics *arstypes.C3Metrics, maxDepth int, avgDepth float64) {
	if maxDepth > metrics.MaxDirectoryDepth {
		metrics.MaxDirectoryDepth = maxDepth
	}
	if avgDepth > metrics.AvgDirectoryDepth {
		metrics.AvgDirectoryDepth = avgDepth
	}
}

// mergeFanoutMetrics computes fanout from an import graph and merges into metrics.
func mergeFanoutMetrics(metrics *arstypes.C3Metrics, graph *shared.ImportGraph) {
	if len(graph.Forward) == 0 {
		return
	}
	totalFanout := 0
	maxFanout := 0
	maxEntity := ""
	for file, deps := range graph.Forward {
		fanout := len(deps)
		totalFanout += fanout
		if fanout > maxFanout {
			maxFanout = fanout
			maxEntity = file
		}
	}
	fanout := arstypes.MetricSummary{
		Avg:       float64(totalFanout) / float64(len(graph.Forward)),
		Max:       maxFanout,
		MaxEntity: maxEntity,
	}
	if fanout.Max > metrics.ModuleFanout.Max {
		metrics.ModuleFanout = fanout
	}
}

// analyzeGoC3 runs Go-specific C3 analysis.
func (a *C3Analyzer) analyzeGoC3() *arstypes.C3Metrics {
	srcPkgs := filterSourcePackages(a.pkgs)

	modulePath := detectModulePath(srcPkgs)
	graph := shared.BuildImportGraph(srcPkgs, modulePath)

	maxDepth, avgDepth := analyzeDirectoryDepth(srcPkgs, modulePath)
	fanout := analyzeModuleFanout(srcPkgs, graph)
	cycles := detectCircularDeps(graph)
	importComp := analyzeImportComplexity(srcPkgs, modulePath)
	deadExports := detectDeadCode(srcPkgs)

	return &arstypes.C3Metrics{
		MaxDirectoryDepth: maxDepth,
		AvgDirectoryDepth: avgDepth,
		ModuleFanout:      fanout,
		CircularDeps:      cycles,
		ImportComplexity:  importComp,
		DeadExports:       deadExports,
	}
}

// filterSourcePackages returns only non-test packages.
func filterSourcePackages(pkgs []*parser.ParsedPackage) []*parser.ParsedPackage {
	var result []*parser.ParsedPackage
	for _, pkg := range pkgs {
		if pkg.ForTest == "" {
			result = append(result, pkg)
		}
	}
	return result
}

// analyzeDirectoryDepth computes the max and average directory depth relative to module root.
// Depth is measured as the number of path segments in the package's relative import path.
// Skips packages outside the module (stdlib, vendor, external deps).
func analyzeDirectoryDepth(pkgs []*parser.ParsedPackage, modulePath string) (int, float64) {
	if len(pkgs) == 0 {
		return 0, 0
	}

	maxDepth := 0
	totalDepth := 0

	for _, pkg := range pkgs {
		depth := packageDepth(pkg.PkgPath, modulePath)
		totalDepth += depth
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	avg := float64(totalDepth) / float64(len(pkgs))
	return maxDepth, avg
}

// packageDepth computes the directory nesting depth of a package relative to its module root.
// Example: "github.com/foo/bar/internal/deep" with module "github.com/foo/bar" -> depth 2
// Depth measures relative path segment count (internal/deep = 2 segments).
func packageDepth(pkgPath, modulePath string) int {
	if pkgPath == modulePath {
		return 0
	}
	rel := strings.TrimPrefix(pkgPath, modulePath+"/")
	if rel == pkgPath {
		return 0
	}
	return len(strings.Split(rel, "/"))
}

// analyzeModuleFanout computes average and max intra-module imports per package.
func analyzeModuleFanout(pkgs []*parser.ParsedPackage, graph *shared.ImportGraph) arstypes.MetricSummary {
	if len(pkgs) == 0 {
		return arstypes.MetricSummary{}
	}

	maxFanout := 0
	maxEntity := ""
	totalFanout := 0

	for _, pkg := range pkgs {
		fanout := len(graph.Forward[pkg.PkgPath])
		totalFanout += fanout
		if fanout > maxFanout {
			maxFanout = fanout
			maxEntity = pkg.PkgPath
		}
	}

	return arstypes.MetricSummary{
		Avg:       float64(totalFanout) / float64(len(pkgs)),
		Max:       maxFanout,
		MaxEntity: maxEntity,
	}
}

// detectCircularDeps uses DFS with node coloring to find dependency cycles.
//
// Algorithm (Tarjan's approach):
// - White (unvisited): Not yet explored
// - Gray (in current DFS path): Currently being explored
// - Black (fully processed): Completed exploration and all descendants
//
// Cycle detection: An edge to a gray node indicates a back-edge (cycle found).
// Cycle reconstruction: Trace parent pointers from current node to gray neighbor.
//
// Note: For Go code, the compiler prevents import cycles, so this returns empty.
// Useful for Python/TypeScript where circular imports are possible.
func detectCircularDeps(graph *shared.ImportGraph) [][]string {
	const (
		white = iota // unvisited
		gray         // in current DFS path
		black        // fully processed
	)

	color := make(map[string]int)
	parent := make(map[string]string)
	var cycles [][]string

	// Initialize all nodes as white.
	for node := range graph.Forward {
		color[node] = white
	}
	// Also add nodes that only appear as targets.
	for _, targets := range graph.Forward {
		for _, t := range targets {
			if _, ok := color[t]; !ok {
				color[t] = white
			}
		}
	}

	var dfs func(node string)
	dfs = func(node string) {
		// Mark node as gray: currently being explored (on the DFS stack)
		color[node] = gray

		for _, neighbor := range graph.Forward[node] {
			switch color[neighbor] {
			case white:
				// Unvisited node: record parent and explore recursively
				parent[neighbor] = node
				dfs(neighbor)
			case gray:
				// Back-edge detected: neighbor is in current DFS path, forming a cycle
				// Trace back from current node to neighbor to reconstruct the cycle
				cycle := []string{neighbor}
				cur := node
				for cur != neighbor {
					cycle = append(cycle, cur)
					cur = parent[cur]
				}
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				cycles = append(cycles, cycle)
			}
		}

		// Mark node as black: fully processed (all descendants explored)
		color[node] = black
	}

	for node := range color {
		if color[node] == white {
			dfs(node)
		}
	}

	return cycles
}

// analyzeImportComplexity computes the maximum import path depth across all packages.
// Depth is measured by segment count (slashes) relative to module root.
// Example: mymodule/internal/analyzer/c1_code_quality has depth=3 (internal, analyzer, c1_code_quality).
func analyzeImportComplexity(pkgs []*parser.ParsedPackage, modulePath string) arstypes.MetricSummary {
	if len(pkgs) == 0 || modulePath == "" {
		return arstypes.MetricSummary{}
	}

	var totalSegments int
	var count int
	maxSegments := 0
	maxEntity := ""

	prefix := modulePath + "/"

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			if !strings.HasPrefix(importPath, prefix) {
				continue
			}
			rel := strings.TrimPrefix(importPath, prefix)
			segments := len(strings.Split(rel, "/"))
			totalSegments += segments
			count++
			if segments > maxSegments {
				maxSegments = segments
				maxEntity = importPath
			}
		}
	}

	if count == 0 {
		return arstypes.MetricSummary{}
	}

	return arstypes.MetricSummary{
		Avg:       float64(totalSegments) / float64(count),
		Max:       maxSegments,
		MaxEntity: maxEntity,
	}
}

// detectDeadCode finds exported functions and types that are never referenced
// by any other package in the module.
//
// Conservative approach:
// - Only flags functions and types (not vars/consts which may be config/constants)
// - Skips main/init (special runtime functions)
// - Skips test packages (test-only exports are valid)
// - Requires multi-package module (single-package projects have no cross-package refs)
//
// Uses go/types TypesInfo.Uses to track cross-package references via type-checker.
// An exported object with no uses from other packages is considered dead.
// exportedSymbol represents a single exported function or type from a package.
type exportedSymbol struct {
	pkg  string
	name string
	file string
	line int
	kind string
	obj  types.Object
}

func detectDeadCode(pkgs []*parser.ParsedPackage) []arstypes.DeadExport {
	if len(pkgs) <= 1 {
		return nil
	}

	exports := collectExports(pkgs)
	crossPkgRef := buildCrossPackageRefs(pkgs)

	var dead []arstypes.DeadExport
	for _, exp := range exports {
		if !crossPkgRef[exp.obj] {
			dead = append(dead, arstypes.DeadExport{
				Package: exp.pkg,
				Name:    exp.name,
				File:    filepath.Base(exp.file),
				Line:    exp.line,
				Kind:    exp.kind,
			})
		}
	}
	return dead
}

// collectExports gathers all exported functions and types from packages.
func collectExports(pkgs []*parser.ParsedPackage) []exportedSymbol {
	var exports []exportedSymbol
	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.TypesInfo == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if sym, ok := classifyExport(pkg, scope.Lookup(name)); ok {
				exports = append(exports, sym)
			}
		}
	}
	return exports
}

// classifyExport checks if an object is an exported func/type and returns its symbol info.
func classifyExport(pkg *parser.ParsedPackage, obj types.Object) (exportedSymbol, bool) {
	if !obj.Exported() {
		return exportedSymbol{}, false
	}
	var kind string
	switch obj.(type) {
	case *types.Func:
		kind = "func"
		if obj.Name() == "main" || obj.Name() == "init" {
			return exportedSymbol{}, false
		}
	case *types.TypeName:
		kind = "type"
	default:
		return exportedSymbol{}, false
	}

	file, line := "", 0
	if pos := obj.Pos(); pos.IsValid() && pkg.Fset != nil {
		position := pkg.Fset.Position(pos)
		file = position.Filename
		line = position.Line
	}
	return exportedSymbol{
		pkg: pkg.PkgPath, name: obj.Name(), file: file, line: line, kind: kind, obj: obj,
	}, true
}

// buildCrossPackageRefs builds a set of objects referenced from a different package.
func buildCrossPackageRefs(pkgs []*parser.ParsedPackage) map[types.Object]bool {
	refs := make(map[types.Object]bool)
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, obj := range pkg.TypesInfo.Uses {
			if obj.Pkg() != nil && obj.Pkg().Path() != pkg.PkgPath {
				refs[obj] = true
			}
		}
	}
	return refs
}

// detectModulePath walks up the directory tree to find go.mod and extracts module path.
// Falls back to common prefix heuristic if go.mod not found or malformed.
// This is used to compute relative package paths for depth and fanout calculations.
func detectModulePath(pkgs []*parser.ParsedPackage) string {
	if len(pkgs) == 0 {
		return ""
	}

	// Try to find go.mod by walking up from the first package's file
	if len(pkgs[0].GoFiles) > 0 {
		dir := filepath.Dir(pkgs[0].GoFiles[0])
		for {
			modFile := filepath.Join(dir, "go.mod")
			data, err := readFile(modFile)
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
		if len(parts) >= modulePathMinPartsC3 {
			return strings.Join(parts[:modulePathMinPartsC3], "/")
		}
		return path
	}
	return ""
}

// readFile reads a file and returns its content.
var readFile = os.ReadFile
