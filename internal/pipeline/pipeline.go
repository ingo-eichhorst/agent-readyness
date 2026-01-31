package pipeline

import (
	"fmt"
	"io"

	"github.com/ingo/agent-readyness/internal/analyzer"
	"github.com/ingo/agent-readyness/internal/discovery"
	"github.com/ingo/agent-readyness/internal/output"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Pipeline orchestrates the scan workflow: discover -> parse -> analyze -> score -> output.
type Pipeline struct {
	verbose    bool
	writer     io.Writer
	parser     Parser
	analyzers  []Analyzer
	scorer     *scoring.Scorer
	results    []*types.AnalysisResult
	scored     *types.ScoredResult
	threshold  float64
	jsonOutput bool
}

// New creates a Pipeline with GoPackagesParser, all three analyzers, and a scorer.
// If cfg is nil, DefaultConfig is used.
func New(w io.Writer, verbose bool, cfg *scoring.ScoringConfig, threshold float64, jsonOutput bool) *Pipeline {
	if cfg == nil {
		cfg = scoring.DefaultConfig()
	}
	return &Pipeline{
		verbose:    verbose,
		writer:     w,
		threshold:  threshold,
		jsonOutput: jsonOutput,
		parser:  &parser.GoPackagesParser{},
		analyzers: []Analyzer{
			&analyzer.C1Analyzer{},
			&analyzer.C3Analyzer{},
			&analyzer.C6Analyzer{},
		},
		scorer: &scoring.Scorer{Config: cfg},
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

	// Stage 3.5: Score results
	scored, err := p.scorer.Score(p.results)
	if err != nil {
		fmt.Fprintf(p.writer, "Warning: scoring error: %v\n", err)
	} else {
		p.scored = scored
	}

	// Stage 4: Render output
	output.RenderSummary(p.writer, result, p.results, p.verbose)
	if p.scored != nil {
		output.RenderScores(p.writer, p.scored, p.verbose)
	}

	return nil
}
