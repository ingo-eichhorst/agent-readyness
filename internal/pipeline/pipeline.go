package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/ingo/agent-readyness/internal/analyzer"
	"github.com/ingo/agent-readyness/internal/discovery"
	"github.com/ingo/agent-readyness/internal/llm"
	"github.com/ingo/agent-readyness/internal/output"
	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/internal/recommend"
	"github.com/ingo/agent-readyness/internal/scoring"
	"github.com/ingo/agent-readyness/pkg/types"
)

// Pipeline orchestrates the scan workflow: discover -> parse -> analyze -> score -> output.
type Pipeline struct {
	verbose      bool
	writer       io.Writer
	parser       Parser
	analyzers    []Analyzer
	c7Analyzer   *analyzer.C7Analyzer // separate for explicit enable control
	scorer       *scoring.Scorer
	results      []*types.AnalysisResult
	scored       *types.ScoredResult
	threshold    float64
	jsonOutput   bool
	onProgress   ProgressFunc
	llmClient    *llm.Client // optional LLM client for C4 analysis
	htmlOutput   string      // optional path for HTML report output
	baselinePath string      // optional path to previous JSON for trend comparison
}

// New creates a Pipeline with GoPackagesParser, all analyzers, and a scorer.
// If cfg is nil, DefaultConfig is used. If onProgress is nil, a no-op is used.
// The pipeline auto-creates a Tree-sitter parser for Python/TypeScript analysis.
func New(w io.Writer, verbose bool, cfg *scoring.ScoringConfig, threshold float64, jsonOutput bool, onProgress ProgressFunc) *Pipeline {
	if cfg == nil {
		cfg = scoring.DefaultConfig()
	}
	if onProgress == nil {
		onProgress = func(string, string) {}
	}

	// Create Tree-sitter parser for Python/TypeScript.
	// If CGO is not enabled or Tree-sitter fails, we continue without it.
	var tsParser *parser.TreeSitterParser
	var tsParserErr error
	tsParser, tsParserErr = parser.NewTreeSitterParser()
	if tsParserErr != nil {
		// Tree-sitter not available; Python/TypeScript analysis will be skipped
		tsParser = nil
	}

	c2Analyzer := analyzer.NewC2Analyzer(tsParser)
	c7Analyzer := analyzer.NewC7Analyzer()

	return &Pipeline{
		verbose:    verbose,
		writer:     w,
		threshold:  threshold,
		jsonOutput: jsonOutput,
		onProgress: onProgress,
		parser:     &parser.GoPackagesParser{},
		analyzers: []Analyzer{
			analyzer.NewC1Analyzer(tsParser),
			c2Analyzer,
			analyzer.NewC3Analyzer(tsParser),
			analyzer.NewC4Analyzer(tsParser),
			analyzer.NewC5Analyzer(), // No tsParser needed - git-based analysis
			analyzer.NewC6Analyzer(tsParser),
			c7Analyzer, // C7 runs but returns Available:false unless enabled
		},
		c7Analyzer: c7Analyzer,
		scorer:     &scoring.Scorer{Config: cfg},
	}
}

// SetLLMClient enables LLM-based analysis for C4 metrics.
func (p *Pipeline) SetLLMClient(client *llm.Client) {
	p.llmClient = client
	// Find and configure C4 analyzer
	for _, a := range p.analyzers {
		if c4, ok := a.(*analyzer.C4Analyzer); ok {
			c4.SetLLMClient(client)
		}
	}
}

// SetC7Enabled enables C7 agent evaluation with the given LLM client (for scoring).
func (p *Pipeline) SetC7Enabled(client *llm.Client) {
	if p.c7Analyzer != nil {
		p.c7Analyzer.Enable(client)
	}
}

// SetHTMLOutput configures HTML report generation.
// If htmlPath is non-empty, an HTML report will be generated at that path.
// If baselinePath is non-empty, the report will include trend comparison.
func (p *Pipeline) SetHTMLOutput(htmlPath, baselinePath string) {
	p.htmlOutput = htmlPath
	p.baselinePath = baselinePath
}

// Run executes the full pipeline on the given directory.
func (p *Pipeline) Run(dir string) error {
	// Stage 1: Discover files
	p.onProgress("discover", "Scanning files...")
	walker := discovery.NewWalker()
	result, err := walker.Discover(dir)
	if err != nil {
		return err
	}

	// Detect project languages
	langs := discovery.DetectProjectLanguages(dir)
	if len(langs) == 0 {
		return fmt.Errorf("no recognized source files found in %s\nSupported languages: Go, Python, TypeScript", dir)
	}

	// Determine which languages have source files
	hasGo := false
	for _, l := range langs {
		if l == types.LangGo {
			hasGo = true
			break
		}
	}

	// Stage 2: Parse Go packages (if Go is present)
	var pkgs []*parser.ParsedPackage
	if hasGo {
		p.onProgress("parse", "Parsing Go packages...")
		pkgs, err = p.parser.Parse(dir)
		if err != nil {
			// Go parsing failed; log warning but continue for other languages
			fmt.Fprintf(p.writer, "Warning: Go parsing error: %v\n", err)
		}
	}

	// Stage 2.5: Build AnalysisTargets for all languages
	var targets []*types.AnalysisTarget

	// Go targets from parsed packages
	if len(pkgs) > 0 {
		goTargets := buildGoTargets(dir, pkgs)
		targets = append(targets, goTargets...)
	}

	// Python and TypeScript targets from discovered files
	nonGoTargets := buildNonGoTargets(dir, result)
	targets = append(targets, nonGoTargets...)

	if len(targets) == 0 {
		return fmt.Errorf("no analyzable source files found in %s", dir)
	}

	// Stage 2.6: Inject Go packages into GoAwareAnalyzers
	if len(pkgs) > 0 {
		for _, a := range p.analyzers {
			if ga, ok := a.(GoAwareAnalyzer); ok {
				ga.SetGoPackages(pkgs)
			}
		}
	}

	// Stage 3: Analyze packages in parallel -- errors are logged but do not abort the scan
	p.onProgress("analyze", "Analyzing code...")
	p.results = nil
	g := new(errgroup.Group)
	var mu sync.Mutex
	var analysisResults []*types.AnalysisResult

	for _, a := range p.analyzers {
		a := a // capture loop variable
		g.Go(func() error {
			ar, err := a.Analyze(targets)
			if err != nil {
				fmt.Fprintf(p.writer, "Warning: %s analyzer error: %v\n", a.Name(), err)
				return nil // don't abort other analyzers
			}
			mu.Lock()
			analysisResults = append(analysisResults, ar)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	// Sort by category name for deterministic output (C1, C2, C3, C4, C5, C6)
	sort.Slice(analysisResults, func(i, j int) bool {
		return analysisResults[i].Category < analysisResults[j].Category
	})
	p.results = analysisResults

	// Stage 3.5: Score results
	p.onProgress("score", "Computing scores...")
	scored, err := p.scorer.Score(p.results)
	if err != nil {
		fmt.Fprintf(p.writer, "Warning: scoring error: %v\n", err)
	} else {
		// Set project name from directory basename
		scored.ProjectName = filepath.Base(dir)
		p.scored = scored
	}

	// Stage 3.6: Generate recommendations
	var recs []recommend.Recommendation
	if p.scored != nil {
		recs = recommend.Generate(p.scored, p.scorer.Config)
	}

	// Stage 4: Render output
	p.onProgress("render", "Generating output...")
	if p.jsonOutput {
		// JSON mode: build report and render as JSON
		if p.scored != nil {
			report := output.BuildJSONReport(p.scored, recs, p.verbose)
			if err := output.RenderJSON(p.writer, report); err != nil {
				return fmt.Errorf("render JSON: %w", err)
			}
		}
	} else {
		// Terminal mode: render summary, scores, then recommendations
		output.RenderSummary(p.writer, result, p.results, p.verbose)
		if p.scored != nil {
			output.RenderScores(p.writer, p.scored, p.verbose)
		}
		if len(recs) > 0 {
			output.RenderRecommendations(p.writer, recs)
		}
	}

	// Stage 4.5: Generate HTML report if requested
	if p.htmlOutput != "" && p.scored != nil {
		if err := p.generateHTMLReport(recs); err != nil {
			return fmt.Errorf("generate HTML report: %w", err)
		}
	}

	// Stage 5: Threshold check (AFTER rendering so output is always displayed)
	if p.threshold > 0 && p.scored != nil && p.scored.Composite < p.threshold {
		return &types.ExitError{
			Code:    2,
			Message: fmt.Sprintf("Score %.1f is below threshold %.1f", p.scored.Composite, p.threshold),
		}
	}

	return nil
}

// generateHTMLReport creates an HTML report file at the configured path.
func (p *Pipeline) generateHTMLReport(recs []recommend.Recommendation) error {
	// Load baseline if provided
	var baseline *types.ScoredResult
	if p.baselinePath != "" {
		var err error
		baseline, err = loadBaseline(p.baselinePath)
		if err != nil {
			// Warn but continue without baseline
			fmt.Fprintf(p.writer, "Warning: could not load baseline: %v\n", err)
		}
	}

	// Create HTML generator
	gen, err := output.NewHTMLGenerator()
	if err != nil {
		return fmt.Errorf("create HTML generator: %w", err)
	}

	// Create output file
	f, err := os.Create(p.htmlOutput)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	// Generate report
	if err := gen.GenerateReport(f, p.scored, recs, baseline); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	return nil
}

// loadBaseline reads a previous JSON output file for trend comparison.
func loadBaseline(path string) (*types.ScoredResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse the JSON report format
	var report output.JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Convert JSONReport to ScoredResult
	result := &types.ScoredResult{
		Composite: report.CompositeScore,
		Tier:      report.Tier,
	}

	for _, cat := range report.Categories {
		result.Categories = append(result.Categories, types.CategoryScore{
			Name:   cat.Name,
			Score:  cat.Score,
			Weight: cat.Weight,
		})
	}

	return result, nil
}

// buildGoTargets creates an []*types.AnalysisTarget from parsed Go packages.
// This bridges the Go-specific parser output to the language-agnostic interface.
func buildGoTargets(rootDir string, pkgs []*parser.ParsedPackage) []*types.AnalysisTarget {
	seen := make(map[string]bool)
	var files []types.SourceFile

	for _, pkg := range pkgs {
		isTest := pkg.ForTest != ""
		for _, goFile := range pkg.GoFiles {
			if seen[goFile] {
				continue
			}
			seen[goFile] = true

			relPath := goFile
			if rel, err := filepath.Rel(rootDir, goFile); err == nil {
				relPath = rel
			}

			class := types.ClassSource
			if isTest {
				class = types.ClassTest
			}

			files = append(files, types.SourceFile{
				Path:     goFile,
				RelPath:  relPath,
				Language: types.LangGo,
				Class:    class,
			})
		}
	}

	if len(files) == 0 {
		return nil
	}

	return []*types.AnalysisTarget{
		{
			Language: types.LangGo,
			RootDir:  rootDir,
			Files:    files,
		},
	}
}

// buildNonGoTargets creates AnalysisTargets for Python and TypeScript from discovered files.
func buildNonGoTargets(rootDir string, scanResult *types.ScanResult) []*types.AnalysisTarget {
	// Group files by language
	langFiles := make(map[types.Language][]types.SourceFile)

	for _, df := range scanResult.Files {
		if df.Language == types.LangGo {
			continue // Go targets built separately from go/packages
		}
		if df.Class == types.ClassExcluded || df.Class == types.ClassGenerated {
			continue
		}

		sf := types.SourceFile{
			Path:     df.Path,
			RelPath:  df.RelPath,
			Language: df.Language,
			Class:    df.Class,
		}

		// Read file content for Tree-sitter (needed during analysis)
		content, err := os.ReadFile(df.Path)
		if err == nil {
			sf.Content = content
			sf.Lines = countFileLines(content)
		}

		langFiles[df.Language] = append(langFiles[df.Language], sf)
	}

	var targets []*types.AnalysisTarget
	for lang, files := range langFiles {
		targets = append(targets, &types.AnalysisTarget{
			Language: lang,
			RootDir:  rootDir,
			Files:    files,
		})
	}

	return targets
}

// countFileLines counts the number of lines in content.
func countFileLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := 1
	for _, b := range content {
		if b == '\n' {
			count++
		}
	}
	return count
}
