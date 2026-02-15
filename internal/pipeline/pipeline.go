package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent"
	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer"
	"github.com/ingo-eichhorst/agent-readyness/internal/discovery"
	"github.com/ingo-eichhorst/agent-readyness/internal/output"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/internal/recommend"
	"github.com/ingo-eichhorst/agent-readyness/internal/scoring"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Pipeline configuration constants.
const (
	evaluatorTimeout = 60 * time.Second // Timeout for CLI-based evaluator
	bytesPerKB       = 1024             // Bytes per kilobyte for file size display
)

// Pipeline orchestrates the scan workflow: discover -> parse -> analyze -> score -> output.
type Pipeline struct {
	verbose      bool
	writer       io.Writer
	parser       parseProvider
	analyzers    []analyzerIface
	c7Analyzer   *analyzer.C7Analyzer // separate for debug features
	scorer       *scoring.Scorer
	results      []*types.AnalysisResult
	scored       *types.ScoredResult
	threshold    float64
	jsonOutput   bool
	onProgress   ProgressFunc
	evaluator    *agent.Evaluator // CLI-based evaluator for LLM analysis
	cliStatus    agent.CLIStatus  // cached CLI availability status
	htmlOutput   string           // optional path for HTML report output
	baselinePath string           // optional path to previous JSON for trend comparison
	badgeOutput  bool             // generate shields.io badge markdown
	debugC7      bool             // C7 debug mode enabled
	debugWriter  io.Writer        // io.Discard (normal) or os.Stderr (debug)
	debugDir     string           // directory for C7 response persistence and replay
	langs        []types.Language // detected project languages
}

// New creates a Pipeline with GoPackagesParser, all analyzers, and a scorer.
// If cfg is nil, DefaultConfig is used. If onProgress is nil, a no-op is used.
// The pipeline auto-creates a Tree-sitter parser for Python/TypeScript analysis.
// CLI availability is detected at startup; if available, LLM features are auto-enabled.
func New(w io.Writer, verbose bool, cfg *scoring.ScoringConfig, threshold float64, jsonOutput bool, onProgress ProgressFunc) *Pipeline {
	if cfg == nil {
		cfg = scoring.DefaultConfig()
	}
	if onProgress == nil {
		onProgress = func(string, string) {}
	}

	tsParser := initTreeSitter()
	c2Analyzer := analyzer.NewC2Analyzer(tsParser)
	c4Analyzer := analyzer.NewC4Analyzer(tsParser)
	c7Analyzer := analyzer.NewC7Analyzer()

	cliStatus, evaluator := initEvaluator(c4Analyzer, c7Analyzer)

	return &Pipeline{
		verbose:     verbose,
		writer:      w,
		threshold:   threshold,
		jsonOutput:  jsonOutput,
		onProgress:  onProgress,
		debugWriter: io.Discard,
		parser:      &parser.GoPackagesParser{},
		analyzers: []analyzerIface{
			analyzer.NewC1Analyzer(tsParser),
			c2Analyzer,
			analyzer.NewC3Analyzer(tsParser),
			c4Analyzer,
			analyzer.NewC5Analyzer(),
			analyzer.NewC6Analyzer(tsParser),
			c7Analyzer,
		},
		c7Analyzer: c7Analyzer,
		scorer:     &scoring.Scorer{Config: cfg},
		evaluator:  evaluator,
		cliStatus:  cliStatus,
	}
}

// initTreeSitter creates a Tree-sitter parser, returning nil if unavailable.
func initTreeSitter() *parser.TreeSitterParser {
	ts, err := parser.NewTreeSitterParser()
	if err != nil {
		return nil
	}
	return ts
}

// initEvaluator detects CLI availability and creates an evaluator if possible.
func initEvaluator(c4 *analyzer.C4Analyzer, c7 *analyzer.C7Analyzer) (agent.CLIStatus, *agent.Evaluator) {
	status := agent.GetCLIStatus()
	if !status.Available {
		return status, nil
	}
	eval := agent.NewEvaluator(evaluatorTimeout)
	c4.SetEvaluator(eval)
	c7.SetEvaluator(eval)
	return status, eval
}

// DisableLLM disables LLM features even when CLI is available.
// This is called when --no-llm flag is set.
func (p *Pipeline) DisableLLM() {
	p.evaluator = nil
	// Find and disable C4 analyzer's LLM evaluation
	for _, a := range p.analyzers {
		if c4, ok := a.(*analyzer.C4Analyzer); ok {
			c4.SetEvaluator(nil)
		}
	}
	// Also disable C7
	if p.c7Analyzer != nil {
		p.c7Analyzer.SetEvaluator(nil)
	}
}

// GetCLIStatus returns the cached CLI availability status.
func (p *Pipeline) GetCLIStatus() agent.CLIStatus {
	return p.cliStatus
}

// SetC7Enabled enables C7 agent evaluation using the CLI-based evaluator.
func (p *Pipeline) SetC7Enabled() {
	if p.c7Analyzer != nil && p.evaluator != nil {
		p.c7Analyzer.Enable(p.evaluator)
	}
}

// SetHTMLOutput configures HTML report generation.
// If htmlPath is non-empty, an HTML report will be generated at that path.
// If baselinePath is non-empty, the report will include trend comparison.
func (p *Pipeline) SetHTMLOutput(htmlPath, baselinePath string) {
	p.htmlOutput = htmlPath
	p.baselinePath = baselinePath
}

// SetBadgeOutput enables shields.io badge markdown generation in output.
func (p *Pipeline) SetBadgeOutput(enabled bool) {
	p.badgeOutput = enabled
}

// SetC7Debug enables C7 debug mode. Debug output goes to stderr via debugWriter.
// This also enables C7 evaluation if not already enabled.
func (p *Pipeline) SetC7Debug(enabled bool) {
	p.debugC7 = enabled
	if enabled {
		p.debugWriter = os.Stderr
	}
	if p.c7Analyzer != nil {
		p.c7Analyzer.SetDebug(enabled, p.debugWriter)
	}
}

// SetDebugDir configures the directory for C7 response persistence and replay.
func (p *Pipeline) SetDebugDir(dir string) {
	p.debugDir = dir
	if p.c7Analyzer != nil {
		p.c7Analyzer.SetDebugDir(dir)
	}
}

// Run executes the full pipeline on the given directory.
func (p *Pipeline) Run(dir string) error {
	result, targets, pkgs, err := p.discoverAndParse(dir)
	if err != nil {
		return err
	}

	p.injectGoPackages(pkgs)

	analysisResults := p.runAnalyzers(targets)
	p.results = analysisResults

	recs := p.scoreAndRecommend(dir)

	if p.debugC7 && p.results != nil {
		output.RenderC7Debug(p.debugWriter, p.results)
	}

	if err := p.renderOutput(result, recs); err != nil {
		return err
	}

	if p.htmlOutput != "" && p.scored != nil {
		if err := p.generateHTMLReport(recs); err != nil {
			return fmt.Errorf("generate HTML report: %w", err)
		}
	}

	if p.threshold > 0 && p.scored != nil && p.scored.Composite < p.threshold {
		return &types.ExitError{
			Code:    2,
			Message: fmt.Sprintf("Score %.1f is below threshold %.1f", p.scored.Composite, p.threshold),
		}
	}

	return nil
}

// discoverAndParse runs discovery, language detection, parsing, and target building.
func (p *Pipeline) discoverAndParse(dir string) (*types.ScanResult, []*types.AnalysisTarget, []*parser.ParsedPackage, error) {
	p.onProgress("discover", "Scanning files...")
	walker := discovery.NewWalker()
	result, err := walker.Discover(dir)
	if err != nil {
		return nil, nil, nil, err
	}

	langs := discovery.DetectProjectLanguages(dir)
	p.langs = langs
	if len(langs) == 0 {
		return nil, nil, nil, fmt.Errorf("no recognized source files found in %s\nSupported languages: Go, Python, TypeScript", dir)
	}

	pkgs := p.parseGoPackages(dir, langs)
	targets := p.buildTargets(dir, result, pkgs)
	if len(targets) == 0 {
		return nil, nil, nil, fmt.Errorf("no analyzable source files found in %s", dir)
	}

	return result, targets, pkgs, nil
}

// parseGoPackages parses Go packages if Go is among the detected languages.
func (p *Pipeline) parseGoPackages(dir string, langs []types.Language) []*parser.ParsedPackage {
	for _, l := range langs {
		if l != types.LangGo {
			continue
		}
		p.onProgress("parse", "Parsing Go packages...")
		pkgs, err := p.parser.Parse(dir)
		if err != nil {
			fmt.Fprintf(p.writer, "Warning: Go parsing error: %v\n", err)
			return nil
		}
		return pkgs
	}
	return nil
}

// buildTargets assembles analysis targets from parsed Go packages and discovered files.
func (p *Pipeline) buildTargets(dir string, result *types.ScanResult, pkgs []*parser.ParsedPackage) []*types.AnalysisTarget {
	var targets []*types.AnalysisTarget
	if len(pkgs) > 0 {
		targets = append(targets, buildGoTargets(dir, pkgs)...)
	}
	targets = append(targets, buildNonGoTargets(dir, result)...)
	return targets
}

// injectGoPackages passes parsed Go packages to analyzers that need them.
func (p *Pipeline) injectGoPackages(pkgs []*parser.ParsedPackage) {
	if len(pkgs) == 0 {
		return
	}
	for _, a := range p.analyzers {
		if ga, ok := a.(goAwareAnalyzer); ok {
			ga.SetGoPackages(pkgs)
		}
	}
}

// runAnalyzers runs all analyzers in parallel and returns sorted results.
func (p *Pipeline) runAnalyzers(targets []*types.AnalysisTarget) []*types.AnalysisResult {
	p.onProgress("analyze", "Analyzing code...")
	g := new(errgroup.Group)
	var mu sync.Mutex
	var results []*types.AnalysisResult

	for _, a := range p.analyzers {
		a := a
		g.Go(func() error {
			ar, err := a.Analyze(targets)
			if err != nil {
				fmt.Fprintf(p.writer, "Warning: %s analyzer error: %v\n", a.Name(), err)
				return nil
			}
			mu.Lock()
			results = append(results, ar)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Category < results[j].Category
	})
	return results
}

// scoreAndRecommend scores analysis results and generates recommendations.
func (p *Pipeline) scoreAndRecommend(dir string) []recommend.Recommendation {
	p.onProgress("score", "Computing scores...")
	scored, err := p.scorer.Score(p.results)
	if err != nil {
		fmt.Fprintf(p.writer, "Warning: scoring error: %v\n", err)
		return nil
	}
	scored.ProjectName = filepath.Base(dir)
	p.scored = scored
	return recommend.Generate(p.scored, p.scorer.Config)
}

// renderOutput renders results in JSON or terminal format.
func (p *Pipeline) renderOutput(result *types.ScanResult, recs []recommend.Recommendation) error {
	p.onProgress("render", "Generating output...")
	if p.jsonOutput {
		if p.scored != nil {
			report := output.BuildJSONReport(p.scored, recs, p.verbose, p.badgeOutput)
			if err := output.RenderJSON(p.writer, report); err != nil {
				return fmt.Errorf("render JSON: %w", err)
			}
		}
		return nil
	}

	output.RenderSummary(p.writer, result, p.results, p.verbose)
	if p.scored != nil {
		output.RenderScores(p.writer, p.scored, p.verbose)
	}
	if len(recs) > 0 {
		output.RenderRecommendations(p.writer, recs)
	}
	if p.badgeOutput && p.scored != nil {
		output.RenderBadge(p.writer, p.scored)
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

	// Build trace data for modal rendering
	langStrings := make([]string, len(p.langs))
	for i, l := range p.langs {
		langStrings[i] = string(l)
	}
	traceData := &output.TraceData{
		ScoringConfig:   p.scorer.Config,
		AnalysisResults: p.results,
		Languages:       langStrings,
	}

	// Generate report
	if err := gen.GenerateReport(f, p.scored, recs, baseline, traceData); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	// Report file size
	if err := f.Sync(); err == nil {
		if fi, err := f.Stat(); err == nil {
			sizeKB := fi.Size() / bytesPerKB
			fmt.Fprintf(p.writer, "HTML report: %s (%d KB)\n", p.htmlOutput, sizeKB)
		}
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

			sf := types.SourceFile{
				Path:     goFile,
				RelPath:  relPath,
				Language: types.LangGo,
				Class:    class,
			}

			// Read file content and count lines (needed for C7 sample selection)
			content, err := os.ReadFile(goFile)
			if err == nil {
				sf.Content = content
				sf.Lines = countFileLines(content)
			}

			files = append(files, sf)
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
