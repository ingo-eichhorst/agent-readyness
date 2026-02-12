package metrics

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ingo/agent-readyness/pkg/types"
)

// M2 sample selection and scoring constants.
const (
	m2SampleCount       = 3                // Number of code samples to evaluate
	m2Timeout           = 360 * time.Second // Total timeout across all samples
	m2MinFileLOC        = 30               // Minimum file size for sample selection
	m2MinComplexity     = 5                // Minimum complexity indicators to qualify
	m2BaseScore         = 2                // Starting score before heuristic adjustments
)

// m2Comprehension measures the agent's ability to understand what code does.
// It tests semantic understanding (behavior), not syntactic correctness.
//
// Research basis: Code comprehension benchmarks show LLMs struggle with
// semantic understanding vs syntactic correctness.
type m2Comprehension struct {
	sampleCount int
	timeout     time.Duration
}

// newM2ComprehensionMetric creates a Code Behavior Comprehension metric.
func newM2ComprehensionMetric() *m2Comprehension {
	return &m2Comprehension{
		sampleCount: m2SampleCount,
		timeout:     m2Timeout,
	}
}

// ID returns the metric identifier.
func (m *m2Comprehension) ID() string { return "code_behavior_comprehension" }

// Name returns the human-readable metric name.
func (m *m2Comprehension) Name() string { return "Code Behavior Comprehension" }

// Description returns what this metric measures.
func (m *m2Comprehension) Description() string {
	return "Measures agent's understanding of what code does (semantics, not syntax)"
}

// Timeout returns the per-metric timeout duration.
func (m *m2Comprehension) Timeout() time.Duration { return m.timeout }

// SampleCount returns the number of samples to evaluate.
func (m *m2Comprehension) SampleCount() int { return m.sampleCount }

// SelectSamples picks complex functions by counting complexity indicators
// (if/for/switch/case statements). Score = complexity_count * (1/sqrt(Lines)).
func (m *m2Comprehension) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []Sample

	// Patterns for complexity indicators across languages
	complexityPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\bif\b`),
		regexp.MustCompile(`\bfor\b`),
		regexp.MustCompile(`\bswitch\b`),
		regexp.MustCompile(`\bcase\b`),
		regexp.MustCompile(`\bwhile\b`),
		regexp.MustCompile(`\btry\b`),
		regexp.MustCompile(`\bcatch\b`),
		regexp.MustCompile(`\belse\b`),
	}

	for _, target := range targets {
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}
			if file.Lines < m2MinFileLOC { // Skip very small files
				continue
			}

			content := string(file.Content)

			// Count complexity indicators
			complexityCount := 0
			for _, pattern := range complexityPatterns {
				matches := pattern.FindAllString(content, -1)
				complexityCount += len(matches)
			}

			if complexityCount < m2MinComplexity { // Skip simple files
				continue
			}

			// Score = complexity / sqrt(lines) - favors dense complexity
			score := float64(complexityCount) / math.Sqrt(float64(file.Lines))

			candidates = append(candidates, Sample{
				FilePath:       file.RelPath,
				SelectionScore: score,
				Description:    fmt.Sprintf("Complex file (%d complexity indicators, %d LOC)", complexityCount, file.Lines),
			})
		}
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SelectionScore > candidates[j].SelectionScore
	})

	if len(candidates) > m.sampleCount {
		candidates = candidates[:m.sampleCount]
	}
	return candidates
}

// comprehensionRubric is the LLM-as-judge rubric for scoring comprehension.
const comprehensionRubric = `You are evaluating an AI coding agent's explanation of code behavior.

The agent was asked to explain what a function or code block does.

Score the response from 1-10 based on these criteria:
- Correctness (50%): Does the explanation accurately describe what the code does?
- Completeness (30%): Does it cover edge cases and error handling paths?
- Clarity (20%): Is the explanation clear and well-structured?

Consider:
- Score 9-10: Correctly explains all paths including edge cases, clear presentation
- Score 7-8: Correct main path, minor gaps in edge cases
- Score 4-6: Partially correct, significant gaps or minor errors
- Score 1-3: Fundamentally misunderstands the code behavior

Respond with JSON only: {"score": N, "reason": "brief explanation"}`

// Execute asks the agent to explain code behavior for each sample.
func (m *m2Comprehension) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

		prompt := fmt.Sprintf(`Read the file at %s and explain what the code does.

Focus on:
1. The main purpose/behavior of the code
2. Important control flow paths (branches, loops)
3. Error handling and edge cases
4. Return values and side effects

Be specific and reference actual code elements.`, sample.FilePath)

		response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read,Grep", timePerSample)
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
			// Heuristic scoring based on response quality indicators
			sr.Score, sr.ScoreTrace = m.scoreComprehensionResponse(response)
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

// scoreComprehensionResponse uses grouped heuristics to score the comprehension explanation.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups: each group contributes +1 if ANY member matches.
// This prevents saturation where many overlapping indicators all score individually.
func (m *m2Comprehension) scoreComprehensionResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)

	trace := ScoreTrace{BaseScore: m2BaseScore}

	// Thematic indicator groups: each group +1 if ANY member matches.
	type indicatorGroup struct {
		name    string
		members []string
	}
	groups := []indicatorGroup{
		{"behavior_understanding", []string{"returns", "return value", "returns the"}},
		{"error_handling", []string{"error", "handles", "handling"}},
		{"control_flow", []string{"if ", "when ", "condition"}},
		{"edge_awareness", []string{"edge case", "corner case", "boundary"}},
		{"side_effects", []string{"side effect", "modifies", "updates"}},
		{"validation", []string{"validates", "checks", "ensures"}},
	}

	for _, group := range groups {
		groupMatched := false
		for _, member := range group.members {
			if strings.Contains(responseLower, member) {
				groupMatched = true
				break
			}
		}
		delta := 0
		if groupMatched {
			delta = 1
		}
		trace.Indicators = append(trace.Indicators, IndicatorMatch{
			Name: "group:" + group.name, Matched: groupMatched, Delta: delta,
		})
	}

	// Negative indicators (superficial or wrong) - individual penalties
	negativeIndicators := []string{
		"i don't know", "unclear", "cannot determine",
		"not sure", "unsure",
	}

	for _, indicator := range negativeIndicators {
		matched := strings.Contains(responseLower, indicator)
		delta := 0
		if matched {
			delta = -1
		}
		trace.Indicators = append(trace.Indicators, IndicatorMatch{
			Name: "negative:" + indicator, Matched: matched, Delta: delta,
		})
	}

	// Hedging penalty group: suggests uncertainty about the explanation
	hedgingIndicators := []string{"might", "probably", "seems to"}
	hedgingMatched := false
	for _, indicator := range hedgingIndicators {
		if strings.Contains(responseLower, indicator) {
			hedgingMatched = true
			break
		}
	}
	hedgingDelta := 0
	if hedgingMatched {
		hedgingDelta = -1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "group:hedging_language", Matched: hedgingMatched, Delta: hedgingDelta,
	})

	// Compute final score from trace
	score := trace.BaseScore
	for _, ind := range trace.Indicators {
		score += ind.Delta
	}
	if score < minScore {
		score = minScore
	}
	if score > maxScore {
		score = maxScore
	}
	trace.FinalScore = score

	return score, trace
}
