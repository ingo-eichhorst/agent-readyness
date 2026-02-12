// Package analyzer provides code analysis implementations for the ARS pipeline.
package c4

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/agent"
	tsp "github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Documentation analysis constants.
const (
	llmAnalysisTimeout  = 5 * time.Minute
	llmCostPerMTok      = 5.0
	readmeTruncateLimit = 20000
	maxExampleFiles     = 3
	exampleContentLimit = 10000
	readmeSummaryLimit  = 5000
	charsPerToken       = 4
	maxSampledFiles     = 5
	toPercentC4         = 100.0
	minCodeBlockMatches = 2
)

// C4Analyzer implements the pipeline.Analyzer interface for C4: Documentation Quality.
// It analyzes README presence, comment density, API doc coverage, and other documentation artifacts.
type C4Analyzer struct {
	tsParser  *tsp.TreeSitterParser
	evaluator *agent.Evaluator // nil if LLM not enabled
}

// NewC4Analyzer creates a C4Analyzer. Tree-sitter parser is needed for Python/TS analysis.
// evaluator can be nil for static-only analysis.
func NewC4Analyzer(tsParser *tsp.TreeSitterParser) *C4Analyzer {
	return &C4Analyzer{tsParser: tsParser, evaluator: nil}
}

// SetEvaluator enables CLI-based content quality evaluation.
func (a *C4Analyzer) SetEvaluator(eval *agent.Evaluator) {
	a.evaluator = eval
}

// Name returns the analyzer display name.
func (a *C4Analyzer) Name() string {
	return "C4: Documentation Quality"
}

// Analyze runs the C4 documentation quality analysis.
//
// Multi-language documentation analysis:
// - Go: Uses go/ast for doc comment detection and go/parser for comment counting
// - Python: Uses Tree-sitter to parse docstrings and comments
// - TypeScript: Uses Tree-sitter for JSDoc and comment analysis
//
// Metrics collected:
// - C4-01: README presence and word count
// - C4-02: Comment density (comment lines / total lines * 100)
// - C4-03: API documentation coverage (documented APIs / total public APIs * 100)
// - C4-04: CHANGELOG presence
// - C4-05: Examples presence (directory or README code blocks)
// - C4-06: CONTRIBUTING guide presence
// - C4-07: Diagrams presence (architecture/design documentation)
//
// Optional LLM-based evaluation (if Claude CLI available):
// - README clarity and completeness
// - Example code quality and usefulness
// - Overall documentation comprehensiveness
func (a *C4Analyzer) Analyze(targets []*types.AnalysisTarget) (*types.AnalysisResult, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets provided")
	}
	rootDir := targets[0].RootDir

	metrics := &types.C4Metrics{
		ChangelogDaysOld: -1, // Default to -1 (not present)
	}

	// C4-01: README presence and word count
	metrics.ReadmePresent, metrics.ReadmeWordCount = analyzeReadme(rootDir)

	// C4-04: CHANGELOG presence
	metrics.ChangelogPresent = analyzeChangelog(rootDir)

	// C4-05: Examples presence
	metrics.ExamplesPresent = analyzeExamples(rootDir)

	// C4-06: CONTRIBUTING presence
	metrics.ContributingPresent = analyzeContributing(rootDir)

	// C4-07: Diagrams presence
	metrics.DiagramsPresent = analyzeDiagrams(rootDir)

	// C4-02 & C4-03: Comment density and API doc coverage across all languages
	totalLines, commentLines := 0, 0
	publicAPIs, documentedAPIs := 0, 0

	for _, target := range targets {
		switch target.Language {
		case types.LangGo:
			tl, cl := analyzeGoComments(target)
			totalLines += tl
			commentLines += cl
			pa, da := analyzeGoAPIDocs(target)
			publicAPIs += pa
			documentedAPIs += da
		case types.LangPython:
			if a.tsParser != nil {
				tl, cl := analyzePythonComments(target, a.tsParser)
				totalLines += tl
				commentLines += cl
				pa, da := analyzePythonAPIDocs(target, a.tsParser)
				publicAPIs += pa
				documentedAPIs += da
			}
		case types.LangTypeScript:
			if a.tsParser != nil {
				tl, cl := analyzeTypeScriptComments(target, a.tsParser)
				totalLines += tl
				commentLines += cl
				pa, da := analyzeTypeScriptAPIDocs(target, a.tsParser)
				publicAPIs += pa
				documentedAPIs += da
			}
		}
	}

	metrics.TotalSourceLines = totalLines
	metrics.CommentLines = commentLines
	metrics.PublicAPIs = publicAPIs
	metrics.DocumentedAPIs = documentedAPIs

	if totalLines > 0 {
		metrics.CommentDensity = float64(commentLines) / float64(totalLines) * toPercentC4
	}
	if publicAPIs > 0 {
		metrics.APIDocCoverage = float64(documentedAPIs) / float64(publicAPIs) * toPercentC4
	}

	// LLM-based content quality evaluation (if enabled)
	if a.evaluator != nil {
		a.runLLMAnalysis(rootDir, metrics)
	}

	// Static metrics are always available (even without LLM)
	metrics.Available = true

	return &types.AnalysisResult{
		Name:     "C4: Documentation Quality",
		Category: "C4",
		Metrics:  map[string]types.CategoryMetrics{"c4": metrics},
	}, nil
}

// runLLMAnalysis performs LLM-based content quality evaluation using Claude CLI.
//
// Optional analysis (requires Claude CLI installed):
// - README clarity: Evaluates readme comprehensiveness and structure
// - Example quality: Assesses example code usefulness and completeness
// - Docs completeness: Overall documentation coverage and gaps
//
// Uses agent.Evaluator with 5-minute timeout and retry logic.
// Gracefully degrades if LLM unavailable (static metrics still provided).
func (a *C4Analyzer) runLLMAnalysis(rootDir string, metrics *types.C4Metrics) {
	metrics.LLMEnabled = true

	ctx, cancel := context.WithTimeout(context.Background(), llmAnalysisTimeout)
	defer cancel()

	totalTokens := 0

	// 1. README clarity evaluation
	if metrics.ReadmePresent {
		readmeContent := readReadmeContent(rootDir)
		if readmeContent != "" {
			eval, err := a.evaluator.EvaluateWithRetry(ctx, agent.ReadmeClarityPrompt, readmeContent)
			if err == nil {
				metrics.ReadmeClarity = eval.Score
				totalTokens += estimateTokens(readmeContent)
			}
		}
	}

	// 2. Example quality evaluation (from README examples or examples directory)
	exampleContent := collectExampleContent(rootDir)
	if exampleContent != "" {
		eval, err := a.evaluator.EvaluateWithRetry(ctx, agent.ExampleQualityPrompt, exampleContent)
		if err == nil {
			metrics.ExampleQuality = eval.Score
			totalTokens += estimateTokens(exampleContent)
		}
	}

	// 3. Completeness evaluation (overall documentation)
	docsContent := collectDocsSummary(rootDir, metrics)
	if docsContent != "" {
		eval, err := a.evaluator.EvaluateWithRetry(ctx, agent.CompletenessPrompt, docsContent)
		if err == nil {
			metrics.Completeness = eval.Score
			totalTokens += estimateTokens(docsContent)
		}
	}

	// 4. Cross-reference coherence evaluation
	if metrics.ReadmePresent {
		readmeContent := readReadmeContent(rootDir)
		if readmeContent != "" {
			eval, err := a.evaluator.EvaluateWithRetry(ctx, agent.CrossRefCoherencePrompt, readmeContent)
			if err == nil {
				metrics.CrossRefCoherence = eval.Score
				totalTokens += estimateTokens(readmeContent)
			}
		}
	}

	// Calculate approximate cost (CLI uses Sonnet pricing, higher than Haiku)
	// Sonnet pricing: ~$3/MTok input, ~$15/MTok output
	// Simplified: ~$0.005 per 1000 tokens blended
	metrics.LLMTokensUsed = totalTokens
	metrics.LLMCostUSD = float64(totalTokens) / 1_000_000 * llmCostPerMTok
	metrics.LLMFilesSampled = countSampledFiles(rootDir)
}

// readReadmeContent reads the README file content.
func readReadmeContent(rootDir string) string {
	readmePaths := []string{
		filepath.Join(rootDir, "README.md"),
		filepath.Join(rootDir, "README"),
		filepath.Join(rootDir, "readme.md"),
		filepath.Join(rootDir, "Readme.md"),
	}

	for _, path := range readmePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			// Truncate very large READMEs to save tokens
			if len(content) > readmeTruncateLimit {
				content = content[:readmeTruncateLimit]
			}
			return string(content)
		}
	}
	return ""
}

// collectExampleContent gathers example code for evaluation.
func collectExampleContent(rootDir string) string {
	var examples strings.Builder

	// Check examples directory
	examplesDirs := []string{
		filepath.Join(rootDir, "examples"),
		filepath.Join(rootDir, "example"),
	}

	for _, dir := range examplesDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		// Collect first few example files
		count := 0
		for _, entry := range entries {
			if entry.IsDir() || count >= maxExampleFiles {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".go" || ext == ".py" || ext == ".ts" || ext == ".js" {
				content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err == nil {
					examples.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", entry.Name(), string(content)))
					count++
				}
			}
		}
		if count > 0 {
			break
		}
	}

	// If no examples dir, extract code blocks from README
	if examples.Len() == 0 {
		readmeContent := readReadmeContent(rootDir)
		codeBlocks := extractCodeBlocks(readmeContent)
		if len(codeBlocks) > 0 {
			for i, block := range codeBlocks {
				if i >= maxExampleFiles {
					break
				}
				examples.WriteString(fmt.Sprintf("=== README Example %d ===\n%s\n\n", i+1, block))
			}
		}
	}

	result := examples.String()
	// Truncate to save tokens
	if len(result) > exampleContentLimit {
		result = result[:exampleContentLimit]
	}
	return result
}

// extractCodeBlocks extracts fenced code blocks from markdown.
func extractCodeBlocks(content string) []string {
	var blocks []string
	inBlock := false
	var currentBlock strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				blocks = append(blocks, currentBlock.String())
				currentBlock.Reset()
				inBlock = false
			} else {
				inBlock = true
			}
		} else if inBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}

	return blocks
}

// collectDocsSummary creates a summary of project documentation for completeness evaluation.
func collectDocsSummary(rootDir string, metrics *types.C4Metrics) string {
	var summary strings.Builder

	summary.WriteString("Documentation Inventory:\n\n")

	if metrics.ReadmePresent {
		summary.WriteString(fmt.Sprintf("- README: Present (%d words)\n", metrics.ReadmeWordCount))
	} else {
		summary.WriteString("- README: NOT PRESENT\n")
	}

	if metrics.ChangelogPresent {
		summary.WriteString("- CHANGELOG: Present\n")
	} else {
		summary.WriteString("- CHANGELOG: NOT PRESENT\n")
	}

	if metrics.ContributingPresent {
		summary.WriteString("- CONTRIBUTING: Present\n")
	} else {
		summary.WriteString("- CONTRIBUTING: NOT PRESENT\n")
	}

	if metrics.ExamplesPresent {
		summary.WriteString("- Examples: Present\n")
	} else {
		summary.WriteString("- Examples: NOT PRESENT\n")
	}

	if metrics.DiagramsPresent {
		summary.WriteString("- Architecture diagrams: Present\n")
	} else {
		summary.WriteString("- Architecture diagrams: NOT PRESENT\n")
	}

	summary.WriteString(fmt.Sprintf("\nCode Statistics:\n"))
	summary.WriteString(fmt.Sprintf("- Comment density: %.1f%%\n", metrics.CommentDensity))
	summary.WriteString(fmt.Sprintf("- API doc coverage: %.1f%%\n", metrics.APIDocCoverage))
	summary.WriteString(fmt.Sprintf("- Public APIs: %d (documented: %d)\n", metrics.PublicAPIs, metrics.DocumentedAPIs))

	// Include README content for context
	readmeContent := readReadmeContent(rootDir)
	if readmeContent != "" {
		// Truncate for token efficiency
		if len(readmeContent) > readmeSummaryLimit {
			readmeContent = readmeContent[:readmeSummaryLimit] + "\n... (truncated)"
		}
		summary.WriteString("\nREADME Content:\n")
		summary.WriteString(readmeContent)
	}

	return summary.String()
}

// estimateTokens provides a rough token count estimate.
func estimateTokens(content string) int {
	// Approximate: 4 characters per token on average
	return len(content) / charsPerToken
}

// countSampledFiles counts files that were sampled for LLM analysis.
func countSampledFiles(rootDir string) int {
	count := 0

	// README
	readmePaths := []string{
		filepath.Join(rootDir, "README.md"),
		filepath.Join(rootDir, "README"),
	}
	for _, path := range readmePaths {
		if _, err := os.Stat(path); err == nil {
			count++
			break
		}
	}

	// Examples
	examplesDirs := []string{
		filepath.Join(rootDir, "examples"),
		filepath.Join(rootDir, "example"),
	}
	for _, dir := range examplesDirs {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					count++
					if count >= maxSampledFiles {
						break
					}
				}
			}
			break
		}
	}

	return count
}

// analyzeReadme checks for README presence and counts words.
//
// Checks multiple naming conventions (case-insensitive):
// - README.md, README, readme.md, Readme.md, README.txt
// - Returns presence flag and word count for scoring
//
// Word counting uses unicode-aware space detection for accuracy across languages.
func analyzeReadme(rootDir string) (bool, int) {
	readmePaths := []string{
		filepath.Join(rootDir, "README.md"),
		filepath.Join(rootDir, "README"),
		filepath.Join(rootDir, "readme.md"),
		filepath.Join(rootDir, "Readme.md"),
		filepath.Join(rootDir, "README.txt"),
	}

	for _, path := range readmePaths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		wordCount := countWords(string(content))
		return true, wordCount
	}

	return false, 0
}

// countWords counts words in text using unicode space detection.
// Accurately handles multi-byte Unicode characters and various whitespace types.
// Returns total word count for README quality metrics.
func countWords(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}
	return words
}

// analyzeChangelog checks for CHANGELOG presence.
// Recognizes common changelog file naming conventions: CHANGELOG, HISTORY, CHANGES.
// Returns boolean indicating whether project maintains a changelog.
func analyzeChangelog(rootDir string) bool {
	changelogPaths := []string{
		filepath.Join(rootDir, "CHANGELOG.md"),
		filepath.Join(rootDir, "CHANGELOG"),
		filepath.Join(rootDir, "changelog.md"),
		filepath.Join(rootDir, "Changelog.md"),
		filepath.Join(rootDir, "HISTORY.md"),
		filepath.Join(rootDir, "CHANGES.md"),
	}

	for _, path := range changelogPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// analyzeExamples checks for examples directory or code blocks in README.
//
// Two detection methods:
// - Filesystem: Checks for examples/, example/, or _examples/ directories
// - README: Counts fenced code blocks (```) in README.md (minimum 2 blocks required)
//
// Returns true if either method finds examples, indicating code samples are provided.
func analyzeExamples(rootDir string) bool {
	// Check for examples/ directory
	examplesDirs := []string{
		filepath.Join(rootDir, "examples"),
		filepath.Join(rootDir, "example"),
		filepath.Join(rootDir, "_examples"),
	}
	for _, dir := range examplesDirs {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return true
		}
	}

	// Check for code blocks in README.md
	readmePath := filepath.Join(rootDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return false
	}

	// Count fenced code blocks (```)
	codeBlockPattern := regexp.MustCompile("(?m)^```")
	matches := codeBlockPattern.FindAllIndex(content, -1)
	// Need at least 2 matches (open and close) for a code block
	return len(matches) >= minCodeBlockMatches
}

// analyzeContributing checks for CONTRIBUTING presence.
func analyzeContributing(rootDir string) bool {
	contributingPaths := []string{
		filepath.Join(rootDir, "CONTRIBUTING.md"),
		filepath.Join(rootDir, "CONTRIBUTING"),
		filepath.Join(rootDir, "contributing.md"),
		filepath.Join(rootDir, ".github", "CONTRIBUTING.md"),
	}

	for _, path := range contributingPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// analyzeDiagrams checks for architecture diagrams in docs/.
func analyzeDiagrams(rootDir string) bool {
	docsDir := filepath.Join(rootDir, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		docsDir = rootDir // Fall back to root if no docs/ dir
	}

	diagramExtensions := []string{".png", ".svg", ".mermaid", ".drawio", ".puml"}
	diagramKeywords := []string{"architecture", "diagram", "flow", "sequence", "class", "er", "uml"}

	found := false
	filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		name := strings.ToLower(filepath.Base(path))

		// Check for diagram file extensions
		for _, diagExt := range diagramExtensions {
			if ext == diagExt {
				// Check if filename contains diagram keywords
				for _, keyword := range diagramKeywords {
					if strings.Contains(name, keyword) {
						found = true
						return nil
					}
				}
			}
		}

		// Also check for mermaid blocks in markdown
		if ext == ".md" {
			content, err := os.ReadFile(path)
			if err == nil && bytes.Contains(content, []byte("```mermaid")) {
				found = true
			}
		}

		return nil
	})

	return found
}

// analyzeGoComments counts total lines and comment lines in Go files.
//
// Comment detection:
// - Line comments: Lines starting with "//" (after trimming whitespace)
// - Block comments: Lines within /* */ pairs (tracks state across lines)
// - Only counts source files (ClassSource), excluding test and generated files
//
// Returns total source lines and comment line count for density calculation.
func analyzeGoComments(target *types.AnalysisTarget) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content, err := os.ReadFile(sf.Path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		inBlockComment := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				commentLines++
			} else if strings.HasPrefix(trimmed, "/*") {
				inBlockComment = true
				commentLines++
			} else if inBlockComment {
				commentLines++
				if strings.Contains(trimmed, "*/") {
					inBlockComment = false
				}
			}
		}
	}
	return
}

// analyzeGoAPIDocs counts exported (public) APIs and those with godoc comments.
//
// API detection:
// - Exported functions: Functions with capitalized names
// - Exported types: Type declarations with capitalized names
// - Documentation check: Presence of Doc comment or associated CommentMap entry
//
// Uses go/ast for precise AST-based analysis of doc comments.
// Returns counts for public API calculation (documented/total).
func analyzeGoAPIDocs(target *types.AnalysisTarget) (publicAPIs, documentedAPIs int) {
	fset := token.NewFileSet()

	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		f, err := goparser.ParseFile(fset, sf.Path, nil, goparser.ParseComments)
		if err != nil {
			continue
		}

		// Build comment map for the file
		cmap := ast.NewCommentMap(fset, f, f.Comments)

		// Check exported functions, types, and methods
		ast.Inspect(f, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				if decl.Name.IsExported() {
					publicAPIs++
					// Check if there's a doc comment
					if decl.Doc != nil && len(decl.Doc.List) > 0 {
						documentedAPIs++
					} else if comments := cmap[decl]; len(comments) > 0 {
						documentedAPIs++
					}
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							publicAPIs++
							// Check for doc comment on the type or containing GenDecl
							if decl.Doc != nil && len(decl.Doc.List) > 0 {
								documentedAPIs++
							} else if s.Doc != nil && len(s.Doc.List) > 0 {
								documentedAPIs++
							}
						}
					}
				}
			}
			return true
		})
	}
	return
}

// analyzePythonComments counts total lines and comment lines in Python files.
func analyzePythonComments(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		// Parse with Tree-sitter to find comments
		tree, err := tsParser.ParseFile(types.LangPython, ".py", content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		commentLines += countTreeSitterCommentLines(root)
		tree.Close()
	}
	return
}

// countTreeSitterCommentLines counts comment lines in a Tree-sitter tree.
func countTreeSitterCommentLines(node *tree_sitter.Node) int {
	count := 0
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if n.Kind() == "comment" {
			// Count lines in the comment
			start := n.StartPosition()
			end := n.EndPosition()
			count += int(end.Row-start.Row) + 1
		}
		for i := uint(0); i < uint(n.ChildCount()); i++ {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}
	walk(node)
	return count
}

// analyzePythonAPIDocs counts public functions/classes and those with docstrings.
func analyzePythonAPIDocs(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (publicAPIs, documentedAPIs int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		tree, err := tsParser.ParseFile(types.LangPython, ".py", content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		pa, da := countPythonAPIDocs(root, content)
		publicAPIs += pa
		documentedAPIs += da
		tree.Close()
	}
	return
}

// countPythonAPIDocs walks the tree to find public functions/classes with docstrings.
func countPythonAPIDocs(root *tree_sitter.Node, content []byte) (publicAPIs, documentedAPIs int) {
	// Walk through direct children of module looking for function_definition and class_definition
	for i := uint(0); i < uint(root.ChildCount()); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "function_definition", "class_definition":
			name := extractPythonDefName(child, content)
			if !strings.HasPrefix(name, "_") { // Public if not starting with _
				publicAPIs++
				if hasPythonDocstring(child) {
					documentedAPIs++
				}
			}
		}
	}
	return
}

// extractPythonDefName extracts the name of a function or class definition.
func extractPythonDefName(node *tree_sitter.Node, content []byte) string {
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "identifier" {
			return child.Utf8Text(content)
		}
	}
	return ""
}

// hasPythonDocstring checks if a function/class has a docstring as its first statement.
func hasPythonDocstring(node *tree_sitter.Node) bool {
	// Look for block child (the body)
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "block" {
			// First non-trivial child of block might be expression_statement with string
			for j := uint(0); j < uint(child.ChildCount()); j++ {
				stmt := child.Child(j)
				if stmt == nil {
					continue
				}
				if stmt.Kind() == "expression_statement" {
					// Check if it contains a string
					for k := uint(0); k < uint(stmt.ChildCount()); k++ {
						expr := stmt.Child(k)
						if expr != nil && expr.Kind() == "string" {
							return true
						}
					}
				}
				// Only check first meaningful statement
				if stmt.Kind() != "comment" {
					return false
				}
			}
			return false
		}
	}
	return false
}

// analyzeTypeScriptComments counts total lines and comment lines in TypeScript files.
func analyzeTypeScriptComments(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (totalLines, commentLines int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		lines := strings.Split(string(content), "\n")
		totalLines += len(lines)

		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := tsParser.ParseFile(types.LangTypeScript, ext, content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		commentLines += countTreeSitterCommentLines(root)
		tree.Close()
	}
	return
}

// analyzeTypeScriptAPIDocs counts exported functions/classes and those with JSDoc.
func analyzeTypeScriptAPIDocs(target *types.AnalysisTarget, tsParser *tsp.TreeSitterParser) (publicAPIs, documentedAPIs int) {
	for _, sf := range target.Files {
		if sf.Class != types.ClassSource {
			continue
		}

		content := sf.Content
		if len(content) == 0 {
			var err error
			content, err = os.ReadFile(sf.Path)
			if err != nil {
				continue
			}
		}

		// Use simpler regex-based approach for export detection
		lines := bytes.Split(content, []byte("\n"))
		inJSDoc := false
		hasJSDoc := false

		for _, line := range lines {
			lineStr := string(bytes.TrimSpace(line))

			// Track JSDoc comments
			if strings.HasPrefix(lineStr, "/**") {
				inJSDoc = true
				hasJSDoc = true
			}
			if inJSDoc && strings.Contains(lineStr, "*/") {
				inJSDoc = false
			}

			// Check for export declarations
			if strings.HasPrefix(lineStr, "export ") {
				if strings.Contains(lineStr, "function ") ||
					strings.Contains(lineStr, "class ") ||
					strings.Contains(lineStr, "const ") ||
					strings.Contains(lineStr, "interface ") ||
					strings.Contains(lineStr, "type ") {
					publicAPIs++
					if hasJSDoc {
						documentedAPIs++
					}
				}
				hasJSDoc = false
			} else if !strings.HasPrefix(lineStr, "*") && !strings.HasPrefix(lineStr, "//") && lineStr != "" {
				// Non-comment, non-export line resets JSDoc tracking
				if !inJSDoc {
					hasJSDoc = false
				}
			}
		}
	}
	return
}
