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
	cfg := createPackageConfig(rootDir)
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	return deduplicateAndConvertPackages(pkgs), nil
}

// createPackageConfig creates a packages.Config for loading Go packages.
func createPackageConfig(rootDir string) *packages.Config {
	return &packages.Config{
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
}

// deduplicateAndConvertPackages converts and deduplicates loaded packages.
func deduplicateAndConvertPackages(pkgs []*packages.Package) []*ParsedPackage {
	seen := make(map[string]*ParsedPackage)
	var result []*ParsedPackage

	for _, pkg := range pkgs {
		if !isValidPackage(pkg) {
			continue
		}

		parsed := convertToParsedPackage(pkg)
		addPackageToResult(parsed, &result, seen)
	}

	return result
}

// isValidPackage checks if a package has valid data or logs errors.
func isValidPackage(pkg *packages.Package) bool {
	if len(pkg.Errors) > 0 {
		for _, e := range pkg.Errors {
			log.Printf("warning: package %s: %s", pkg.PkgPath, e)
		}
		// Still include if we got useful data (partial results)
		if pkg.Types == nil || len(pkg.Syntax) == 0 {
			return false
		}
	}
	return true
}

// convertToParsedPackage converts a packages.Package to ParsedPackage.
func convertToParsedPackage(pkg *packages.Package) *ParsedPackage {
	return &ParsedPackage{
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
}

// addPackageToResult adds a package to the result, handling deduplication.
func addPackageToResult(parsed *ParsedPackage, result *[]*ParsedPackage, seen map[string]*ParsedPackage) {
	// Test packages (ForTest != "") are always added
	if parsed.ForTest != "" {
		*result = append(*result, parsed)
		return
	}

	// Source packages: keep only one per PkgPath
	if _, exists := seen[parsed.PkgPath]; !exists {
		seen[parsed.PkgPath] = parsed
		*result = append(*result, parsed)
	}
}
