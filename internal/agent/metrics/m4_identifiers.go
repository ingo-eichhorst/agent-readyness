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
	candidates := m4CollectCandidates(targets)

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	unique := m4Deduplicate(candidates)
	return m4ConvertToSamples(unique, m.sampleCount)
}

var (
	m4GoExportedFunc = regexp.MustCompile(`(?m)^func\s+([A-Z][a-zA-Z0-9_]*)\s*\(`)
	m4GoExportedType = regexp.MustCompile(`(?m)^type\s+([A-Z][a-zA-Z0-9_]*)\s+`)
	m4GoExportedVar  = regexp.MustCompile(`(?m)^(?:var|const)\s+([A-Z][a-zA-Z0-9_]*)`)
	m4TsExported     = regexp.MustCompile(`(?m)^export\s+(?:function|class|const|let|var|interface|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	m4PyPublic       = regexp.MustCompile(`(?m)^(?:def|class)\s+([a-zA-Z][a-zA-Z0-9_]*)`)
)

func m4PatternsForLang(lang types.Language) []*regexp.Regexp {
	switch lang {
	case types.LangGo:
		return []*regexp.Regexp{m4GoExportedFunc, m4GoExportedType, m4GoExportedVar}
	case types.LangTypeScript:
		return []*regexp.Regexp{m4TsExported}
	case types.LangPython:
		return []*regexp.Regexp{m4PyPublic}
	default:
		return nil
	}
}

func m4CollectCandidates(targets []*types.AnalysisTarget) []identifierCandidate {
	var candidates []identifierCandidate
	for _, target := range targets {
		patterns := m4PatternsForLang(target.Language)
		if patterns == nil {
			continue
		}
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}
			candidates = append(candidates, m4ExtractFromFile(file, patterns)...)
		}
	}
	return candidates
}

func m4ExtractFromFile(file types.SourceFile, patterns []*regexp.Regexp) []identifierCandidate {
	content := string(file.Content)
	var candidates []identifierCandidate
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(content, -1)
		for _, match := range matches {
			if len(match) < 4 {
				continue
			}
			name := content[match[2]:match[3]]
			if len(name) < m4MinNameLength {
				continue
			}
			nameLower := strings.ToLower(name)
			if strings.HasPrefix(nameLower, "test") ||
				strings.HasPrefix(nameLower, "example") ||
				strings.HasPrefix(nameLower, "mock") {
				continue
			}
			lineNum := 1 + strings.Count(content[:match[0]], "\n")
			wordCount := countIdentifierWords(name)
			candidates = append(candidates, identifierCandidate{
				name:     name,
				filePath: file.RelPath,
				line:     lineNum,
				score:    float64(len(name)) * float64(wordCount),
			})
		}
	}
	return candidates
}

func m4Deduplicate(candidates []identifierCandidate) []identifierCandidate {
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

func m4ConvertToSamples(unique []identifierCandidate, maxCount int) []Sample {
	var samples []Sample
	for i, c := range unique {
		if i >= maxCount {
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
	return executeStandardMetric(ctx, workDir, samples, executor, executeConfig{
		metricID:   m.ID(),
		metricName: m.Name(),
		timeout:    m.timeout,
		tools:      "Read",
		buildPrompt: func(sample Sample) string {
			return fmt.Sprintf(`Without reading the file, interpret what the identifier "%s" means based ONLY on its name.

1. What is the likely purpose of this identifier?
2. What type of thing is it (function, type, variable, constant)?
3. What domain/concern does it belong to?

After your interpretation, read %s (line %d) to verify your interpretation.

Format:
- Interpretation: [your interpretation based on name alone]
- Verification: [what you found in the code]
- Accuracy: [how accurate was your interpretation?]`, sample.FunctionName, sample.FilePath, sample.StartLine)
		},
		scoreResponse: m.scoreIdentifierResponse,
	})
}

// scoreIdentifierResponse uses grouped heuristics to score the identifier interpretation.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups with variable weights. Self-report groups have
// higher impact (+2/-2) since the agent's own accuracy assessment is a strong signal.
func (m *m4Identifiers) scoreIdentifierResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)

	trace := ScoreTrace{BaseScore: m4BaseScore}

	// Self-report positive group (+2): agent confirms its interpretation was right
	matchedPositive := strings.Contains(responseLower, "accurate")
	deltaPositive := 0
	if matchedPositive {
		deltaPositive = m4SelfReportPositiveDelta
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "group:self_report_positive", Matched: matchedPositive, Delta: deltaPositive,
	})

	// Self-report partial/negative groups with custom deltas
	groups := matchGroups(responseLower, []indicatorGroup{
		{"self_report_partial", []string{"mostly correct", "partially"}},
		{"self_report_negative", []string{"incorrect", "wrong", "misunderstood"}},
		{"detailed_interpretation", []string{"interpretation:", "purpose:"}},
		{"action_words", []string{"handles", "manages", "creates", "processes", "returns", "validates", "converts", "parses"}},
		{"structure_verification", []string{"verification:"}},
		{"structure_accuracy", []string{"accuracy:"}},
	})
	// Override delta for self_report_negative to -2
	for i := range groups {
		if groups[i].Name == "group:self_report_negative" && groups[i].Matched {
			groups[i].Delta = m4SelfReportNegativeDelta
		}
	}
	trace.Indicators = append(trace.Indicators, groups...)

	return computeScore(&trace), trace
}
