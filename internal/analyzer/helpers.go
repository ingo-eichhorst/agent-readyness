// Package analyzer provides code analysis implementations for the ARS pipeline.
package analyzer

import (
	"strings"

	"github.com/ingo/agent-readyness/internal/parser"
)

// ImportGraph holds forward and reverse adjacency lists for intra-module imports.
type ImportGraph struct {
	Forward map[string][]string // package -> packages it imports (efferent)
	Reverse map[string][]string // package -> packages that import it (afferent)
}

// BuildImportGraph constructs an import graph from parsed packages, filtering
// to only intra-module imports (those with the given module path prefix).
func BuildImportGraph(pkgs []*parser.ParsedPackage, modulePath string) *ImportGraph {
	g := &ImportGraph{
		Forward: make(map[string][]string),
		Reverse: make(map[string][]string),
	}

	for _, pkg := range pkgs {
		if pkg.ForTest != "" {
			continue // skip test packages for coupling analysis
		}
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, modulePath) {
				g.Forward[pkg.PkgPath] = append(g.Forward[pkg.PkgPath], importPath)
				g.Reverse[importPath] = append(g.Reverse[importPath], pkg.PkgPath)
			}
		}
	}

	return g
}
