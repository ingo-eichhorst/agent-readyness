package pipeline

import (
	"fmt"
	"io"

	"github.com/ingo/agent-readyness/internal/analyzer"
	"github.com/ingo/agent-readyness/internal/discovery"
	"github.com/ingo/agent-readyness/internal/output"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Pipeline orchestrates the scan workflow: discover -> parse -> analyze -> output.
type Pipeline struct {
	verbose   bool
	writer    io.Writer
	parser    Parser
	analyzers []Analyzer
	results   []*types.AnalysisResult
}

// New creates a Pipeline with GoPackagesParser and all three analyzers.
func New(w io.Writer, verbose bool) *Pipeline {
	return &Pipeline{
		verbose: verbose,
		writer:  w,
		parser:  &parser.GoPackagesParser{},
		analyzers: []Analyzer{
			&analyzer.C1Analyzer{},
			&analyzer.C3Analyzer{},
			&analyzer.C6Analyzer{},
		},
	}
}

// Run executes the full pipeline on the given directory.
func (p *Pipeline) Run(dir string) error {
	// Stage 1: Discover files
	walker := discovery.NewWalker()
	result, err := walker.Discover(dir)
	if err != nil {
		return err
	}

	// Stage 2: Parse packages (loads ASTs, type info, imports via go/packages)
	pkgs, err := p.parser.Parse(dir)
	if err != nil {
		return err
	}

	// Stage 3: Analyze packages -- errors are logged but do not abort the scan
	p.results = nil
	for _, a := range p.analyzers {
		ar, err := a.Analyze(pkgs)
		if err != nil {
			fmt.Fprintf(p.writer, "Warning: %s analyzer error: %v\n", a.Name(), err)
			continue
		}
		p.results = append(p.results, ar)
	}

	// Stage 4: Render output
	output.RenderSummary(p.writer, result, p.results, p.verbose)

	return nil
}
