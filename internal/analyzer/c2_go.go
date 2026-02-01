package analyzer

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C2GoAnalyzer computes C2 (Semantic Explicitness) metrics for Go code using go/ast.
type C2GoAnalyzer struct {
	pkgs []*parser.ParsedPackage
}

// Analyze computes C2 metrics for a Go AnalysisTarget.
func (a *C2GoAnalyzer) Analyze(target *types.AnalysisTarget) (*types.C2LanguageMetrics, error) {
	// Filter to source packages only (skip test packages)
	var srcPkgs []*parser.ParsedPackage
	for _, pkg := range a.pkgs {
		if pkg.ForTest != "" {
			continue
		}
		srcPkgs = append(srcPkgs, pkg)
	}

	if len(srcPkgs) == 0 {
		return &types.C2LanguageMetrics{
			TypeAnnotationCoverage: 100,
			TypeStrictness:         1,
		}, nil
	}

	metrics := &types.C2LanguageMetrics{
		// Go is statically typed with compile-time type checking
		TypeAnnotationCoverage: 100,
		TypeStrictness:         1,
	}

	// Count total LOC
	for _, pkg := range srcPkgs {
		for _, f := range pkg.Syntax {
			metrics.LOC += pkg.Fset.Position(f.End()).Line
		}
	}

	// C2-GO-01: interface{}/any usage rate (NullSafety metric)
	anyUsage := analyzeAnyUsage(srcPkgs)
	metrics.NullSafety = anyUsage.safetyPercent

	// C2-GO-02: Naming consistency
	naming := analyzeNamingConsistency(srcPkgs)
	metrics.NamingConsistency = naming.consistencyPercent
	metrics.TotalIdentifiers = naming.totalChecked

	// C2-GO-03: Magic numbers
	magic := analyzeMagicNumbers(srcPkgs)
	metrics.MagicNumberCount = magic.count
	if metrics.LOC > 0 {
		metrics.MagicNumberRatio = float64(magic.count) / float64(metrics.LOC) * 1000
	}

	// C2-GO-04: Nil safety patterns
	nilSafety := analyzeNilSafety(srcPkgs)
	// Blend interface{}/any safety with nil safety (equal weight)
	if nilSafety.totalPointerUsages > 0 {
		nilSafetyPercent := float64(nilSafety.checkedUsages) / float64(nilSafety.totalPointerUsages) * 100
		if nilSafetyPercent > 100 {
			nilSafetyPercent = 100
		}
		// Average of any-safety and nil-safety
		metrics.NullSafety = (anyUsage.safetyPercent + nilSafetyPercent) / 2
	}

	// Count total functions
	for _, pkg := range srcPkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncDecl); ok {
					metrics.TotalFunctions++
				}
				return true
			})
		}
	}

	return metrics, nil
}

// anyUsageResult holds interface{}/any usage analysis results.
type anyUsageResult struct {
	totalTypeRefs  int
	anyRefs        int
	safetyPercent  float64 // 100 - anyUsagePercent
}

// analyzeAnyUsage counts interface{}/any usage across all source packages.
func analyzeAnyUsage(pkgs []*parser.ParsedPackage) anyUsageResult {
	var result anyUsageResult

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.InterfaceType:
					result.totalTypeRefs++
					// Empty interface: Methods.List is nil or empty
					if node.Methods == nil || len(node.Methods.List) == 0 {
						result.anyRefs++
					}
				case *ast.Ident:
					if node.Name == "any" {
						// Check if this is the builtin 'any' type (not a local variable named 'any')
						if node.Obj == nil {
							result.totalTypeRefs++
							result.anyRefs++
						}
					}
				}
				return true
			})
		}
	}

	if result.totalTypeRefs == 0 {
		result.safetyPercent = 100 // No type refs means no unsafe usage
	} else {
		anyPercent := float64(result.anyRefs) / float64(result.totalTypeRefs) * 100
		result.safetyPercent = 100 - anyPercent
		if result.safetyPercent < 0 {
			result.safetyPercent = 0
		}
	}

	return result
}

// namingResult holds naming consistency analysis results.
type namingResult struct {
	totalChecked       int
	consistent         int
	consistencyPercent float64
}

// commonAcronyms are uppercase abbreviations that are valid in Go identifiers.
var commonAcronyms = map[string]bool{
	"ID": true, "URL": true, "HTTP": true, "HTTPS": true,
	"API": true, "JSON": true, "XML": true, "SQL": true,
	"HTML": true, "CSS": true, "DNS": true, "EOF": true,
	"IP": true, "TCP": true, "UDP": true, "TLS": true,
	"SSL": true, "SSH": true, "RPC": true, "CPU": true,
	"GPU": true, "RAM": true, "ROM": true, "OS": true,
	"IO": true, "DB": true, "UI": true, "OK": true,
}

// analyzeNamingConsistency checks Go naming conventions.
func analyzeNamingConsistency(pkgs []*parser.ParsedPackage) namingResult {
	var result namingResult

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			// Check function declarations
			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					if node.Name == nil {
						return true
					}
					name := node.Name.Name
					if shouldSkipName(name) {
						return true
					}
					result.totalChecked++
					if isConsistentGoName(name, ast.IsExported(name)) {
						result.consistent++
					}

				case *ast.TypeSpec:
					if node.Name == nil {
						return true
					}
					name := node.Name.Name
					if shouldSkipName(name) {
						return true
					}
					result.totalChecked++
					if isConsistentGoName(name, ast.IsExported(name)) {
						result.consistent++
					}

				case *ast.ValueSpec:
					for _, ident := range node.Names {
						name := ident.Name
						if shouldSkipName(name) {
							continue
						}
						result.totalChecked++
						if isConsistentGoName(name, ast.IsExported(name)) {
							result.consistent++
						}
					}
				}
				return true
			})
		}
	}

	if result.totalChecked == 0 {
		result.consistencyPercent = 100
	} else {
		result.consistencyPercent = float64(result.consistent) / float64(result.totalChecked) * 100
	}

	return result
}

// shouldSkipName returns true if the identifier should be excluded from naming checks.
func shouldSkipName(name string) bool {
	// Skip blank identifier
	if name == "_" {
		return true
	}
	// Skip single-letter variables
	if len(name) <= 1 {
		return true
	}
	// Skip common acronyms that are standalone identifiers
	if commonAcronyms[name] {
		return true
	}
	return false
}

// isConsistentGoName checks if a name follows Go naming conventions.
// Exported names should start with uppercase (CamelCase).
// Unexported names should start with lowercase (camelCase).
func isConsistentGoName(name string, exported bool) bool {
	if len(name) == 0 {
		return true
	}

	firstRune := rune(name[0])

	if exported {
		// Must start with uppercase
		if !unicode.IsUpper(firstRune) {
			return false
		}
	} else {
		// Must start with lowercase
		if !unicode.IsLower(firstRune) && firstRune != '_' {
			return false
		}
	}

	// Check for snake_case (underscores in the middle) -- not idiomatic Go
	if strings.Contains(name[1:], "_") {
		// Allow underscores in test functions (Test_xxx) and common patterns
		if !strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "test") {
			return false
		}
	}

	return true
}

// magicNumberResult holds magic number analysis results.
type magicNumberResult struct {
	count int
}

// analyzeMagicNumbers counts magic numbers outside const blocks.
func analyzeMagicNumbers(pkgs []*parser.ParsedPackage) magicNumberResult {
	var result magicNumberResult

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				// Skip const declarations
				if genDecl, ok := n.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
					return false // Don't walk into const blocks
				}

				lit, ok := n.(*ast.BasicLit)
				if !ok {
					return true
				}

				if lit.Kind != token.INT && lit.Kind != token.FLOAT {
					return true
				}

				// Exclude 0, 1, -1
				if isCommonNumericLiteral(lit.Value) {
					return true
				}

				result.count++
				return true
			})
		}
	}

	return result
}

// isCommonNumericLiteral returns true for 0, 1, and other universally understood constants.
func isCommonNumericLiteral(value string) bool {
	switch value {
	case "0", "1", "0.0", "1.0", "2", "0x0", "0x1":
		return true
	}
	return false
}

// nilSafetyResult holds nil safety analysis results.
type nilSafetyResult struct {
	totalPointerUsages int
	checkedUsages      int
}

// analyzeNilSafety computes a simple nil-check-to-pointer-usage ratio.
func analyzeNilSafety(pkgs []*parser.ParsedPackage) nilSafetyResult {
	var result nilSafetyResult

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			// Count nil checks and pointer dereferences per function
			ast.Inspect(f, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					return true
				}

				var nilChecks int
				var pointerUsages int

				ast.Inspect(fn.Body, func(inner ast.Node) bool {
					switch node := inner.(type) {
					case *ast.BinaryExpr:
						// Count nil comparisons (x == nil or x != nil)
						if (node.Op == token.EQL || node.Op == token.NEQ) && isNilIdent(node.Y) {
							nilChecks++
						}
						if (node.Op == token.EQL || node.Op == token.NEQ) && isNilIdent(node.X) {
							nilChecks++
						}
					case *ast.StarExpr:
						// Pointer dereference (excluding type expressions)
						if _, isType := node.X.(*ast.Ident); isType {
							pointerUsages++
						}
					}
					return true
				})

				result.totalPointerUsages += pointerUsages
				// Each nil check "covers" one pointer usage
				covered := nilChecks
				if covered > pointerUsages {
					covered = pointerUsages
				}
				result.checkedUsages += covered

				return false // Don't recurse into nested functions from here
			})
		}
	}

	return result
}

// isNilIdent returns true if the expression is the nil identifier.
func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}
