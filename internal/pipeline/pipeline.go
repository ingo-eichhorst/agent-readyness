package pipeline

import (
	"io"

	"github.com/ingo/agent-readyness/internal/discovery"
	"github.com/ingo/agent-readyness/internal/output"
)

// Pipeline orchestrates the scan workflow: discover -> parse -> analyze -> output.
type Pipeline struct {
	verbose   bool
	writer    io.Writer
	parser    Parser
	analyzers []Analyzer
}

// New creates a Pipeline with default stub parser and analyzer.
func New(w io.Writer, verbose bool) *Pipeline {
	return &Pipeline{
		verbose:   verbose,
		writer:    w,
		parser:    &StubParser{},
		analyzers: []Analyzer{&StubAnalyzer{}},
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

	// Stage 3: Analyze packages
	for _, analyzer := range p.analyzers {
		_, err := analyzer.Analyze(pkgs)
		if err != nil {
			return err
		}
	}

	// Stage 4: Render output
	output.RenderSummary(p.writer, result, p.verbose)

	return nil
}
