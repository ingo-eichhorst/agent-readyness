package metrics

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ingo/agent-readyness/pkg/types"
)

// M4Identifiers measures the agent's ability to infer meaning from identifier names.
// It tests semantic interpretation of naming conventions without surrounding context.
//
// Research basis: Descriptive compound identifiers improve comprehension;
// this tests the agent's ability to leverage meaningful naming.
type M4Identifiers struct {
	sampleCount int
	timeout     time.Duration
}

// NewM4IdentifiersMetric creates an Identifier Interpretability metric.
func NewM4IdentifiersMetric() *M4Identifiers {
	return &M4Identifiers{
		sampleCount: 5,
		timeout:     60 * time.Second,
	}
}

func (m *M4Identifiers) ID() string { return "identifier_interpretability" }
func (m *M4Identifiers) Name() string { return "Identifier Interpretability" }
func (m *M4Identifiers) Description() string {
	return "Measures ability to infer meaning from identifier names"
}
func (m *M4Identifiers) Timeout() time.Duration { return m.timeout }
func (m *M4Identifiers) SampleCount() int { return m.sampleCount }

// identifierCandidate holds an extracted identifier and its source.
type identifierCandidate struct {
	name     string
	filePath string
	line     int
	score    float64 // Selection score based on name length/complexity
}

// SelectSamples extracts exported identifiers and selects those with longer,
// more semantically rich names. Longer names = more semantic content to test.
func (m *M4Identifiers) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []identifierCandidate

	// Patterns for exported identifiers by language
	goExportedFunc := regexp.MustCompile(`(?m)^func\s+([A-Z][a-zA-Z0-9_]*)\s*\(`)
	goExportedType := regexp.MustCompile(`(?m)^type\s+([A-Z][a-zA-Z0-9_]*)\s+`)
	goExportedVar := regexp.MustCompile(`(?m)^(?:var|const)\s+([A-Z][a-zA-Z0-9_]*)`)
	tsExported := regexp.MustCompile(`(?m)^export\s+(?:function|class|const|let|var|interface|type)\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	pyPublic := regexp.MustCompile(`(?m)^(?:def|class)\s+([a-zA-Z][a-zA-Z0-9_]*)`) // Python uses convention, not export

	for _, target := range targets {
		var patterns []*regexp.Regexp

		switch target.Language {
		case types.LangGo:
			patterns = []*regexp.Regexp{goExportedFunc, goExportedType, goExportedVar}
		case types.LangTypeScript:
			patterns = []*regexp.Regexp{tsExported}
		case types.LangPython:
			patterns = []*regexp.Regexp{pyPublic}
		default:
			continue
		}

		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}

			content := string(file.Content)
			lines := strings.Split(content, "\n")

			for _, pattern := range patterns {
				matches := pattern.FindAllStringSubmatchIndex(content, -1)
				for _, match := range matches {
					if len(match) >= 4 {
						name := content[match[2]:match[3]]

						// Skip very short names (less semantic content)
						if len(name) < 4 {
							continue
						}

						// Skip obvious test/example identifiers
						nameLower := strings.ToLower(name)
						if strings.HasPrefix(nameLower, "test") ||
							strings.HasPrefix(nameLower, "example") ||
							strings.HasPrefix(nameLower, "mock") {
							continue
						}

						// Calculate line number
						lineNum := 1 + strings.Count(content[:match[0]], "\n")

						// Score based on name length and word count (camelCase/snake_case words)
						wordCount := countIdentifierWords(name)
						score := float64(len(name)) * float64(wordCount)

						candidates = append(candidates, identifierCandidate{
							name:     name,
							filePath: file.RelPath,
							line:     lineNum,
							score:    score,
						})
					}
				}
			}
			_ = lines // Used above for reference
		}
	}

	// Sort by score descending (longer, more complex names first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Deduplicate by name (keep highest scored occurrence)
	seen := make(map[string]bool)
	var unique []identifierCandidate
	for _, c := range candidates {
		if !seen[c.name] {
			seen[c.name] = true
			unique = append(unique, c)
		}
	}

	// Convert to samples
	var samples []Sample
	for i, c := range unique {
		if i >= m.sampleCount {
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
func (m *M4Identifiers) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

// scoreIdentifierResponse uses heuristics to score the identifier interpretation.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
func (m *M4Identifiers) scoreIdentifierResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)

	trace := ScoreTrace{BaseScore: 5}

	// Check for self-reported accuracy (agent verifies its own interpretation)
	matchedAccurate := strings.Contains(responseLower, "accurate") || strings.Contains(responseLower, "correct")
	deltaAccurate := 0
	if matchedAccurate {
		deltaAccurate = 2
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "self_report:accurate/correct", Matched: matchedAccurate, Delta: deltaAccurate,
	})

	matchedPartial := strings.Contains(responseLower, "mostly correct") || strings.Contains(responseLower, "partially")
	deltaPartial := 0
	if matchedPartial {
		deltaPartial = 1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "self_report:mostly_correct/partially", Matched: matchedPartial, Delta: deltaPartial,
	})

	matchedWrong := strings.Contains(responseLower, "incorrect") || strings.Contains(responseLower, "wrong") || strings.Contains(responseLower, "misunderstood")
	deltaWrong := 0
	if matchedWrong {
		deltaWrong = -2
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "self_report:incorrect/wrong/misunderstood", Matched: matchedWrong, Delta: deltaWrong,
	})

	// Positive indicators (detailed interpretation)
	positiveIndicators := []string{
		"interpretation:", "purpose:", "function", "type",
		"handles", "manages", "creates", "processes",
		"returns", "validates", "converts", "parses",
	}

	for _, indicator := range positiveIndicators {
		matched := strings.Contains(responseLower, indicator)
		delta := 0
		if matched {
			delta = 1
		}
		trace.Indicators = append(trace.Indicators, IndicatorMatch{
			Name: "positive:" + indicator, Matched: matched, Delta: delta,
		})
	}

	// Structure check (response has expected sections)
	matchedVerification := strings.Contains(responseLower, "verification:")
	deltaVerification := 0
	if matchedVerification {
		deltaVerification = 1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "structure:verification", Matched: matchedVerification, Delta: deltaVerification,
	})

	matchedAccuracySection := strings.Contains(responseLower, "accuracy:")
	deltaAccuracySection := 0
	if matchedAccuracySection {
		deltaAccuracySection = 1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "structure:accuracy", Matched: matchedAccuracySection, Delta: deltaAccuracySection,
	})

	// Compute final score from trace
	score := trace.BaseScore
	for _, ind := range trace.Indicators {
		score += ind.Delta
	}
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}
	trace.FinalScore = score

	return score, trace
}
