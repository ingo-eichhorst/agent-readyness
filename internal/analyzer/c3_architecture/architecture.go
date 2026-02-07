package c3

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/ingo/agent-readyness/internal/analyzer/shared"
	"github.com/ingo/agent-readyness/internal/parser"
	arstypes "github.com/ingo/agent-readyness/pkg/types"
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

	// Go analysis (existing logic)
	if a.pkgs != nil {
		goMetrics := a.analyzeGoC3()
		metrics = goMetrics
	}

	// Python/TypeScript analysis via targets
	for _, target := range targets {
		var analysis *languageAnalysis
		var err error

		switch target.Language {
		case arstypes.LangPython:
			analysis, err = a.analyzeLanguageTarget(
				target,
				pyFilterSourceFiles,
				pyBuildImportGraph,
				pyDetectDeadCode,
				pyAnalyzeDirectoryDepth,
			)
		case arstypes.LangTypeScript:
			analysis, err = a.analyzeLanguageTarget(
				target,
				tsFilterSourceFiles,
				tsBuildImportGraph,
				tsDetectDeadCode,
				tsAnalyzeDirectoryDepth,
			)
		}

		if err == nil {
			mergeLanguageAnalysis(metrics, analysis)
		}
	}

	return &arstypes.AnalysisResult{
		Name:     "C3: Architecture",
		Category: "C3",
		Metrics:  map[string]interface{}{"c3": metrics},
	}, nil
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
// e.g., "github.com/foo/bar/internal/deep" with module "github.com/foo/bar" -> depth 2
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

// detectCircularDeps uses DFS with white/gray/black coloring to find cycles in the import graph.
// In valid Go code, the compiler prevents import cycles, so this returns empty for compilable code.
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
		color[node] = gray

		for _, neighbor := range graph.Forward[node] {
			switch color[neighbor] {
			case white:
				parent[neighbor] = node
				dfs(neighbor)
			case gray:
				// Found a cycle: trace back from node to neighbor.
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

		color[node] = black
	}

	for node := range color {
		if color[node] == white {
			dfs(node)
		}
	}

	return cycles
}

// analyzeImportComplexity computes the average number of path segments in intra-module imports.
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

// detectDeadCode finds exported functions and types that are never referenced by any other package.
// Conservative: only flags functions and types (not vars/consts), skips main/init, skips test packages.
func detectDeadCode(pkgs []*parser.ParsedPackage) []arstypes.DeadExport {
	type exportedSymbol struct {
		pkg  string
		name string
		file string
		line int
		kind string
		obj  types.Object
	}

	var exports []exportedSymbol

	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.TypesInfo == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}

			var kind string
			switch obj.(type) {
			case *types.Func:
				kind = "func"
				if name == "main" || name == "init" {
					continue
				}
			case *types.TypeName:
				kind = "type"
			default:
				continue // Skip vars and consts.
			}

			pos := obj.Pos()
			file := ""
			line := 0
			if pos.IsValid() && pkg.Fset != nil {
				position := pkg.Fset.Position(pos)
				file = position.Filename
				line = position.Line
			}

			exports = append(exports, exportedSymbol{
				pkg:  pkg.PkgPath,
				name: name,
				file: file,
				line: line,
				kind: kind,
				obj:  obj,
			})
		}
	}

	// Build cross-package reference set: objects referenced from a different package.
	crossPkgRef := make(map[types.Object]bool)
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		for _, obj := range pkg.TypesInfo.Uses {
			if obj.Pkg() != nil && obj.Pkg().Path() != pkg.PkgPath {
				crossPkgRef[obj] = true
			}
		}
	}

	// Flag exports with no cross-package reference.
	var dead []arstypes.DeadExport
	for _, exp := range exports {
		if crossPkgRef[exp.obj] {
			continue
		}

		// For single-package modules, skip (no cross-package possible).
		if len(pkgs) <= 1 {
			continue
		}

		dead = append(dead, arstypes.DeadExport{
			Package: exp.pkg,
			Name:    exp.name,
			File:    filepath.Base(exp.file),
			Line:    exp.line,
			Kind:    exp.kind,
		})
	}

	return dead
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
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/")
		}
		return path
	}
	return ""
}

// readFile reads a file and returns its content.
var readFile = os.ReadFile
