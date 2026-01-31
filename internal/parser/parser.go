// Package parser provides Go package loading using go/packages for type-aware
// AST parsing, type information, and import graph resolution.
package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

// ParsedPackage holds all analysis-relevant data for a single Go package
// loaded via go/packages.
type ParsedPackage struct {
	ID        string                        // Unique package identifier from go/packages
	Name      string                        // Package name (e.g., "main", "parser")
	PkgPath   string                        // Full import path
	GoFiles   []string                      // .go source file paths
	Syntax    []*ast.File                   // Parsed AST for each file
	Fset      *token.FileSet                // Shared file set for position info
	Types     *types.Package                // Type-checked package
	TypesInfo *types.Info                   // Detailed type info (uses, defs, etc.)
	Imports   map[string]*packages.Package  // Direct imports (path -> package)
	ForTest   string                        // Non-empty if this is a test package
}

// GoPackagesParser loads Go packages from a module directory using go/packages.
type GoPackagesParser struct{}

// Parse loads all packages in the given root directory using go/packages.Load.
// It returns source packages and test packages separately identified via ForTest.
// Packages with errors are skipped with a log warning.
func (p *GoPackagesParser) Parse(rootDir string) ([]*ParsedPackage, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedForTest,
		Dir:   rootDir,
		Tests: true,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	// Deduplicate by PkgPath. For each PkgPath, prefer the non-test variant
	// for source analysis. Test variants (ForTest != "") are kept separately.
	// This avoids the go/packages test-package duplication issue.
	seen := make(map[string]*ParsedPackage)
	var result []*ParsedPackage

	for _, pkg := range pkgs {
		// Skip packages with errors but log them
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				log.Printf("warning: package %s: %s", pkg.PkgPath, e)
			}
			// Still include if we got useful data (partial results)
			if pkg.Types == nil || len(pkg.Syntax) == 0 {
				continue
			}
		}

		parsed := &ParsedPackage{
			ID:        pkg.ID,
			Name:      pkg.Name,
			PkgPath:   pkg.PkgPath,
			GoFiles:   pkg.GoFiles,
			Syntax:    pkg.Syntax,
			Fset:      pkg.Fset,
			Types:     pkg.Types,
			TypesInfo: pkg.TypesInfo,
			Imports:   pkg.Imports,
			ForTest:   pkg.ForTest,
		}

		// Deduplication: if this is a test package, always add it
		// (test packages have ForTest set). For source packages,
		// keep only one per PkgPath.
		if pkg.ForTest != "" {
			result = append(result, parsed)
		} else {
			if _, exists := seen[pkg.PkgPath]; !exists {
				seen[pkg.PkgPath] = parsed
				result = append(result, parsed)
			}
		}
	}

	return result, nil
}
