package metrics

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// M4 sample selection and scoring constants.
const (
	m4SampleCount   = 5                // Number of identifiers to evaluate
	m4Timeout       = 180 * time.Second // Total timeout across all samples
	m4MinNameLength = 4                // Minimum identifier name length for selection
	m4BaseScore     = 1                // Starting score before heuristic adjustments
	m4SelfReportPositiveDelta  = 2     // Delta for self-reported accurate interpretation
	m4SelfReportNegativeDelta  = -2    // Delta for self-reported incorrect interpretation
)

// m4Identifiers measures the agent's ability to infer meaning from identifier names.
// It tests semantic interpretation of naming conventions without surrounding context.
//
// Research basis: Descriptive compound identifiers improve comprehension;
// this tests the agent's ability to leverage meaningful naming.
type m4Identifiers struct {
	sampleCount int
	timeout     time.Duration
}

// newM4IdentifiersMetric creates an Identifier Interpretability metric.
func newM4IdentifiersMetric() *m4Identifiers {
	return &m4Identifiers{
		sampleCount: m4SampleCount,
		timeout:     m4Timeout,
	}
}

// ID returns the metric identifier.
func (m *m4Identifiers) ID() string { return "identifier_interpretability" }

// Name returns the human-readable metric name.
func (m *m4Identifiers) Name() string { return "Identifier Interpretability" }

// Description returns what this metric measures.
func (m *m4Identifiers) Description() string {
	return "Measures ability to infer meaning from identifier names"
}

// Timeout returns the per-metric timeout duration.
func (m *m4Identifiers) Timeout() time.Duration { return m.timeout }

// SampleCount returns the number of samples to evaluate.
func (m *m4Identifiers) SampleCount() int { return m.sampleCount }

// identifierCandidate holds an extracted identifier and its source.
type identifierCandidate struct {
	name     string
	filePath string
	line     int
	score    float64 // Selection score based on name length/complexity
}

// SelectSamples extracts exported identifiers and selects those with longer,
// more semantically rich names. Longer names = more semantic content to test.
func (m *m4Identifiers) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	candidates := m4ExtractCandidates(targets)
	candidates = m4SortAndDeduplicate(candidates)
	return m4ConvertToSamples(candidates, m.sampleCount)
}

func m4ExtractCandidates(targets []*types.AnalysisTarget) []identifierCandidate {
	var candidates []identifierCandidate
	patterns := m4CompilePatterns()

	for _, target := range targets {
		targetPatterns := m4SelectPatternsForLanguage(patterns, target.Language)
		if targetPatterns == nil {
			continue
		}
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}
			m4ExtractFromFile(file, targetPatterns, &candidates)
		}
	}
	return candidates
}

type m4Patterns struct {
	goFunc *regexp.Regexp
	goType *regexp.Regexp
	goVar  *regexp.Regexp
	ts     *regexp.Regexp
	py     *regexp.Regexp
}

func m4CompilePatterns() m4Patterns {
	return m4Patterns{
		goFunc: regexp.MustCompile(`(?m)^func\s+([A-Z][a-zA-Z0-9_]*)\s*\(`),
		goType: regexp.MustCompile(`(?m)^type\s+([A-Z][a-zA-Z0-9_]*)\s+`),
		goVar:  regexp.MustCompile(`(?m)^(?:var|const)\s+([A-Z][a-zA-Z0-9_]*)`),
		ts:     regexp.MustCompile(`(?m)^export\s+(?:function|class|const|let|var|interface|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		py:     regexp.MustCompile(`(?m)^(?:def|class)\s+([a-zA-Z][a-zA-Z0-9_]*)`),
	}
}

func m4SelectPatternsForLanguage(patterns m4Patterns, lang types.Language) []*regexp.Regexp {
	switch lang {
	case types.LangGo:
		return []*regexp.Regexp{patterns.goFunc, patterns.goType, patterns.goVar}
	case types.LangTypeScript:
		return []*regexp.Regexp{patterns.ts}
	case types.LangPython:
		return []*regexp.Regexp{patterns.py}
	default:
		return nil
	}
}

func m4ExtractFromFile(file types.SourceFile, patterns []*regexp.Regexp, candidates *[]identifierCandidate) {
	content := string(file.Content)
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			name := content[match[2]:match[3]]
			if !m4IsValidIdentifier(name) {
				continue
			}
			lineNum := 1 + strings.Count(content[:match[0]], "\n")
			wordCount := countIdentifierWords(name)
			score := float64(len(name)) * float64(wordCount)
			*candidates = append(*candidates, identifierCandidate{
				name:     name,
				filePath: file.RelPath,
				line:     lineNum,
				score:    score,
			})
		}
	}
}

func m4IsValidIdentifier(name string) bool {
	if len(name) < m4MinNameLength {
		return false
	}
	nameLower := strings.ToLower(name)
	return !strings.HasPrefix(nameLower, "test") &&
		!strings.HasPrefix(nameLower, "example") &&
		!strings.HasPrefix(nameLower, "mock")
}

func m4SortAndDeduplicate(candidates []identifierCandidate) []identifierCandidate {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	seen := make(map[string]bool)
	var unique []identifierCandidate
	for _, c := range candidates {
		if !seen[c.name] {
			seen[c.name] = true
			unique = append(unique, c)
		}
	}
	return unique
}

func m4ConvertToSamples(candidates []identifierCandidate, sampleCount int) []Sample {
	var samples []Sample
	for i, c := range candidates {
		if i >= sampleCount {
			break
		}
		samples = append(samples, Sample{
			FilePath:       c.filePath,
			FunctionName:   c.name,
			StartLine:      c.line,
			SelectionScore: c.score,
			Description:    fmt.Sprintf("Exported identifier '%s' with %d chars", c.name, len(c.name)),
		})
	}
	return samples
}

// countIdentifierWords counts "words" in an identifier (camelCase or snake_case).
func countIdentifierWords(name string) int {
	// Split by underscore
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")
		count := 0
		for _, p := range parts {
			if len(p) > 0 {
				count++
			}
		}
		return count
	}

	// Count camelCase transitions
	count := 1
	for i := 1; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			count++
		}
	}
	return count
}

// identifiersRubric is the LLM-as-judge rubric for scoring identifier interpretation.
const identifiersRubric = `You are evaluating an AI coding agent's interpretation of an identifier name.

The agent was asked to infer the purpose of an identifier from its name alone.

Score the response from 1-10 based on these criteria:
- Correctness (60%): Does the interpretation match the actual purpose?
- Specificity (40%): Is the interpretation detailed and precise?

Consider:
- Score 9-10: Correctly interprets the purpose with specific details
- Score 7-8: Mostly correct, captures main purpose
- Score 4-6: Partially correct but vague or incomplete
- Score 1-3: Misinterprets the identifier meaning

Respond with JSON only: {"score": N, "reason": "brief explanation"}`

// Execute asks the agent to interpret identifier meanings.
func (m *m4Identifiers) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
	result := MetricResult{
		MetricID:   m.ID(),
		MetricName: m.Name(),
	}
	startTime := time.Now()

	if len(samples) == 0 {
		result.Error = "no samples available for evaluation"
		result.Duration = time.Since(startTime)
		return result
	}

	timePerSample := m.timeout / time.Duration(len(samples))
	var sampleResults []SampleResult
	var totalScore int
	successCount := 0

	for _, sample := range samples {
		sampleStart := time.Now()
		sampleCtx, cancel := context.WithTimeout(ctx, timePerSample)

		// Provide identifier name and file context for verification
		prompt := fmt.Sprintf(`Without reading the file, interpret what the identifier "%s" means based ONLY on its name.

1. What is the likely purpose of this identifier?
2. What type of thing is it (function, type, variable, constant)?
3. What domain/concern does it belong to?

After your interpretation, read %s (line %d) to verify your interpretation.

Format:
- Interpretation: [your interpretation based on name alone]
- Verification: [what you found in the code]
- Accuracy: [how accurate was your interpretation?]`, sample.FunctionName, sample.FilePath, sample.StartLine)

		response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read", timePerSample)
		cancel()

		sr := SampleResult{
			Sample:   sample,
			Response: response,
			Prompt:   prompt,
			Duration: time.Since(sampleStart),
		}

		if err != nil {
			sr.Error = err.Error()
			sr.Score = 0
		} else {
			sr.Score, sr.ScoreTrace = m.scoreIdentifierResponse(response)
			totalScore += sr.Score
			successCount++
		}

		sampleResults = append(sampleResults, sr)
	}

	result.Samples = sampleResults
	result.Duration = time.Since(startTime)

	if successCount == 0 {
		result.Score = 0
		result.Error = "all samples failed"
		return result
	}

	result.Score = totalScore / successCount
	return result
}

// scoreIdentifierResponse uses grouped heuristics to score the identifier interpretation.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups with variable weights. Self-report groups have
// higher impact (+2/-2) since the agent's own accuracy assessment is a strong signal.
func (m *m4Identifiers) scoreIdentifierResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)
	trace := ScoreTrace{BaseScore: m4BaseScore}

	groups := []indicatorGroup{
		{name: "group:self_report_positive", patterns: []string{"accurate"}, delta: m4SelfReportPositiveDelta},
		{name: "group:self_report_partial", patterns: []string{"mostly correct", "partially"}, delta: 1},
		{name: "group:self_report_negative", patterns: []string{"incorrect", "wrong", "misunderstood"}, delta: m4SelfReportNegativeDelta},
		{name: "group:detailed_interpretation", patterns: []string{"interpretation:", "purpose:"}, delta: 1},
		{name: "group:action_words", patterns: []string{"handles", "manages", "creates", "processes", "returns", "validates", "converts", "parses"}, delta: 1},
		{name: "group:structure_verification", patterns: []string{"verification:"}, delta: 1},
		{name: "group:structure_accuracy", patterns: []string{"accuracy:"}, delta: 1},
	}
	checkGroups(&trace, responseLower, groups)

	return computeTraceScore(&trace), trace
}
