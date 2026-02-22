package c3

import (
	"path/filepath"

	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// codeDefinition represents a named definition found during dead code analysis.
type codeDefinition struct {
	Name string
	File string
	Line int
	Kind string
}

// deadCodeExtractor is implemented by language-specific dead code analyzers.
type deadCodeExtractor interface {
	collectDefinitions(files []*parser.ParsedTreeSitterFile) []codeDefinition
	collectImportedNames(files []*parser.ParsedTreeSitterFile) map[string]bool
}

// detectTreeSitterDeadCode runs the shared 3-step dead code algorithm using a language-specific extractor.
func detectTreeSitterDeadCode(files []*parser.ParsedTreeSitterFile, extractor deadCodeExtractor) []types.DeadExport {
	if len(files) <= 1 {
		return nil
	}
	defs := extractor.collectDefinitions(files)
	names := extractor.collectImportedNames(files)
	return flagDeadExports(defs, names)
}

// flagDeadExports returns definitions not found in the imported names set.
func flagDeadExports(defs []codeDefinition, importedNames map[string]bool) []types.DeadExport {
	var dead []types.DeadExport
	for _, d := range defs {
		if !importedNames[d.Name] {
			dead = append(dead, types.DeadExport{
				Package: "",
				Name:    d.Name,
				File:    filepath.Base(d.File),
				Line:    d.Line,
				Kind:    d.Kind,
			})
		}
	}
	return dead
}
