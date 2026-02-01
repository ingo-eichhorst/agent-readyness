// Package analyzer provides code analysis implementations for the ARS pipeline.
package analyzer

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
	"golang.org/x/tools/cover"
)

// C6Analyzer implements the pipeline.Analyzer interface for C6: Testing Infrastructure.
// It measures test detection, test-to-code ratio, coverage parsing,
// test isolation, and assertion density.
// It also implements GoAwareAnalyzer for Go-specific analysis via SetGoPackages.
type C6Analyzer struct {
	pkgs []*parser.ParsedPackage
}

// Name returns the analyzer display name.
func (a *C6Analyzer) Name() string {
	return "C6: Testing"
}

// SetGoPackages stores Go-specific parsed packages for use during Analyze.
func (a *C6Analyzer) SetGoPackages(pkgs []*parser.ParsedPackage) {
	a.pkgs = pkgs
}

// Analyze runs all 5 C6 sub-metrics over the parsed packages.
func (a *C6Analyzer) Analyze(_ []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	pkgs := a.pkgs
	// Separate test packages from source packages
	var srcPkgs, testPkgs []*parser.ParsedPackage
	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			testPkgs = append(testPkgs, pkg)
		} else {
			srcPkgs = append(srcPkgs, pkg)
		}
	}

	metrics := &types.C6Metrics{}

	// C6-01: Test Detection
	metrics.TestFileCount = countFiles(testPkgs)
	metrics.SourceFileCount = countFiles(srcPkgs)

	// C6-02: Test-to-Code Ratio
	metrics.TestToCodeRatio = calculateTestRatio(srcPkgs, testPkgs)

	// C6-03: Coverage Parsing
	// Derive rootDir from first package's files
	rootDir := deriveRootDir(pkgs)
	if rootDir != "" {
		pct, src, err := a.parseCoverage(rootDir)
		if err != nil {
			return nil, fmt.Errorf("coverage parsing: %w", err)
		}
		metrics.CoveragePercent = pct
		metrics.CoverageSource = src
	} else {
		metrics.CoveragePercent = -1
		metrics.CoverageSource = "none"
	}

	// C6-04: Test Isolation
	metrics.TestIsolation = analyzeIsolation(testPkgs, metrics)

	// C6-05: Assertion Density
	analyzeAssertions(testPkgs, metrics)

	return &types.AnalysisResult{
		Name:     "C6: Testing",
		Category: "C6",
		Metrics: map[string]interface{}{
			"c6": metrics,
		},
	}, nil
}

// countFiles counts the total number of .go files across packages.
func countFiles(pkgs []*parser.ParsedPackage) int {
	count := 0
	for _, pkg := range pkgs {
		for _, f := range pkg.GoFiles {
			if strings.HasSuffix(f, ".go") {
				count++
			}
		}
	}
	return count
}

// calculateTestRatio calculates test LOC / source LOC.
func calculateTestRatio(srcPkgs, testPkgs []*parser.ParsedPackage) float64 {
	srcLOC := countLOC(srcPkgs)
	testLOC := countLOC(testPkgs)

	if srcLOC == 0 {
		return 0
	}
	return float64(testLOC) / float64(srcLOC)
}

// countLOC counts total lines of code across packages using fset positions.
func countLOC(pkgs []*parser.ParsedPackage) int {
	total := 0
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			if pkg.Fset != nil {
				endPos := pkg.Fset.Position(f.End())
				total += endPos.Line
			}
		}
	}
	return total
}

// deriveRootDir derives the project root directory from the first package's files.
func deriveRootDir(pkgs []*parser.ParsedPackage) string {
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) > 0 {
			// Walk up from the file to find go.mod
			dir := filepath.Dir(pkg.GoFiles[0])
			for dir != "/" && dir != "." {
				if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
					return dir
				}
				dir = filepath.Dir(dir)
			}
			// Fallback: return the first file's directory
			return filepath.Dir(pkg.GoFiles[0])
		}
	}
	return ""
}

// parseCoverage tries to parse coverage files in the given directory.
// Search order: cover.out (Go native), lcov.info/coverage.lcov (LCOV), cobertura.xml/coverage.xml (Cobertura).
// Returns coverage percentage, source identifier, and any error.
func (a *C6Analyzer) parseCoverage(dir string) (float64, string, error) {
	// Try Go cover profile
	for _, name := range []string{"cover.out"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			pct, err := parseGoCoverage(path)
			if err != nil {
				return -1, "none", fmt.Errorf("go coverage: %w", err)
			}
			return pct, "go-cover", nil
		}
	}

	// Try LCOV
	for _, name := range []string{"lcov.info", "coverage.lcov"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			pct, err := parseLCOV(path)
			if err != nil {
				return -1, "none", fmt.Errorf("lcov: %w", err)
			}
			return pct, "lcov", nil
		}
	}

	// Try Cobertura
	for _, name := range []string{"cobertura.xml", "coverage.xml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			pct, err := parseCobertura(path)
			if err != nil {
				return -1, "none", fmt.Errorf("cobertura: %w", err)
			}
			return pct, "cobertura", nil
		}
	}

	return -1, "none", nil
}

// parseGoCoverage parses a Go coverage profile using x/tools/cover.
func parseGoCoverage(path string) (float64, error) {
	profiles, err := cover.ParseProfiles(path)
	if err != nil {
		return 0, err
	}

	var totalStatements, coveredStatements int
	for _, p := range profiles {
		for _, block := range p.Blocks {
			totalStatements += block.NumStmt
			if block.Count > 0 {
				coveredStatements += block.NumStmt
			}
		}
	}

	if totalStatements == 0 {
		return 0, nil
	}
	return float64(coveredStatements) / float64(totalStatements) * 100, nil
}

// parseLCOV parses an LCOV format coverage file.
func parseLCOV(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var totalLines, hitLines int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DA:") {
			parts := strings.SplitN(line[3:], ",", 2)
			if len(parts) == 2 {
				totalLines++
				if parts[1] != "0" {
					hitLines++
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	if totalLines == 0 {
		return 0, nil
	}
	return float64(hitLines) / float64(totalLines) * 100, nil
}

// coberturaXML represents the top-level coverage element in a Cobertura report.
type coberturaXML struct {
	XMLName  xml.Name `xml:"coverage"`
	LineRate float64  `xml:"line-rate,attr"`
}

// parseCobertura parses a Cobertura XML coverage file.
func parseCobertura(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var report coberturaXML
	if err := xml.Unmarshal(data, &report); err != nil {
		return 0, err
	}

	return report.LineRate * 100, nil
}

// externalDepPackages are packages that indicate a test has external dependencies.
var externalDepPackages = map[string]bool{
	"net/http":     true,
	"net":          true,
	"database/sql": true,
	"os/exec":      true,
	"net/rpc":      true,
	"net/smtp":     true,
}

// analyzeIsolation checks test packages for external dependency imports.
// Returns percentage of isolated tests (0-100).
func analyzeIsolation(testPkgs []*parser.ParsedPackage, metrics *types.C6Metrics) float64 {
	totalTests := 0
	isolatedTests := 0

	for _, pkg := range testPkgs {
		// Collect file-level imports
		fileImports := make(map[string]map[string]bool) // filename -> set of import paths
		for _, f := range pkg.Syntax {
			fname := ""
			if pkg.Fset != nil {
				fname = pkg.Fset.Position(f.Pos()).Filename
			}
			imports := make(map[string]bool)
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				imports[path] = true
			}
			fileImports[fname] = imports
		}

		// Walk AST to find test functions
		for _, f := range pkg.Syntax {
			fname := ""
			if pkg.Fset != nil {
				fname = pkg.Fset.Position(f.Pos()).Filename
			}
			imports := fileImports[fname]

			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					continue
				}
				if !isTestFunction(fn) {
					continue
				}

				totalTests++
				hasExtDep := hasExternalDep(imports)

				if !hasExtDep {
					isolatedTests++
				}
			}
		}
	}

	if totalTests == 0 {
		return 100 // No tests = vacuously isolated
	}
	return float64(isolatedTests) / float64(totalTests) * 100
}

// isTestFunction checks if a function declaration is a test function.
// Test functions start with "Test" and have a single *testing.T parameter.
func isTestFunction(fn *ast.FuncDecl) bool {
	if !strings.HasPrefix(fn.Name.Name, "Test") {
		return false
	}
	// Check for *testing.T parameter
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	param := fn.Type.Params.List[0]
	starExpr, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selExpr, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selExpr.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "testing" && selExpr.Sel.Name == "T"
}

// hasExternalDep checks if any import is in the external dependency list.
func hasExternalDep(imports map[string]bool) bool {
	for imp := range imports {
		if externalDepPackages[imp] {
			return true
		}
		// Also flag common third-party HTTP/DB packages
		if strings.Contains(imp, "github.com") && (strings.Contains(imp, "http") || strings.Contains(imp, "sql") || strings.Contains(imp, "redis") || strings.Contains(imp, "mongo")) {
			return true
		}
	}
	return false
}

// Standard testing assertion methods
var stdAssertionMethods = map[string]bool{
	"Error":   true,
	"Errorf":  true,
	"Fatal":   true,
	"Fatalf":  true,
	"Fail":    true,
	"FailNow": true,
}

// Testify-style assertion methods
var testifyAssertionMethods = map[string]bool{
	"Equal":    true,
	"NotEqual": true,
	"True":     true,
	"False":    true,
	"Nil":      true,
	"NotNil":   true,
	"Contains": true,
	"NoError":  true,
	"Len":      true,
	"Empty":    true,
	"Greater":  true,
	"Less":     true,
	"ErrorIs":  true,
	"ErrorAs":  true,
}

// analyzeAssertions counts assertions per test function and populates metrics.
func analyzeAssertions(testPkgs []*parser.ParsedPackage, metrics *types.C6Metrics) {
	var totalAssertions int
	maxAssertions := 0
	maxEntity := ""

	for _, pkg := range testPkgs {
		for _, f := range pkg.Syntax {
			fname := ""
			if pkg.Fset != nil {
				fname = pkg.Fset.Position(f.Pos()).Filename
			}

			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					continue
				}
				if !isTestFunction(fn) {
					continue
				}

				count := countAssertions(fn)
				totalAssertions += count

				tfm := types.TestFunctionMetric{
					Package:        pkg.PkgPath,
					Name:           fn.Name.Name,
					File:           fname,
					AssertionCount: count,
				}
				if pkg.Fset != nil {
					tfm.Line = pkg.Fset.Position(fn.Pos()).Line
				}

				metrics.TestFunctions = append(metrics.TestFunctions, tfm)

				if count > maxAssertions {
					maxAssertions = count
					maxEntity = fn.Name.Name
				}
			}
		}
	}

	numFuncs := len(metrics.TestFunctions)
	if numFuncs > 0 {
		metrics.AssertionDensity = types.MetricSummary{
			Avg:       float64(totalAssertions) / float64(numFuncs),
			Max:       maxAssertions,
			MaxEntity: maxEntity,
		}
	}
}

// countAssertions counts assertion method calls within a test function body.
func countAssertions(fn *ast.FuncDecl) int {
	if fn.Body == nil {
		return 0
	}

	count := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name

		// Check standard testing methods (t.Error, t.Fatal, etc.)
		if stdAssertionMethods[methodName] {
			count++
			return true
		}

		// Check testify-style assertion methods (assert.Equal, require.NoError, etc.)
		if testifyAssertionMethods[methodName] {
			count++
			return true
		}

		return true
	})

	return count
}
