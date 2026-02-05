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

// M2Comprehension measures the agent's ability to understand what code does.
// It tests semantic understanding (behavior), not syntactic correctness.
//
// Research basis: Code comprehension benchmarks show LLMs struggle with
// semantic understanding vs syntactic correctness.
type M2Comprehension struct {
	sampleCount int
	timeout     time.Duration
}

// NewM2ComprehensionMetric creates a Code Behavior Comprehension metric.
func NewM2ComprehensionMetric() *M2Comprehension {
	return &M2Comprehension{
		sampleCount: 3,
		timeout:     120 * time.Second,
	}
}

func (m *M2Comprehension) ID() string { return "code_behavior_comprehension" }
func (m *M2Comprehension) Name() string { return "Code Behavior Comprehension" }
func (m *M2Comprehension) Description() string {
	return "Measures agent's understanding of what code does (semantics, not syntax)"
}
func (m *M2Comprehension) Timeout() time.Duration { return m.timeout }
func (m *M2Comprehension) SampleCount() int { return m.sampleCount }

// SelectSamples picks complex functions by counting complexity indicators
// (if/for/switch/case statements). Score = complexity_count * (1/sqrt(Lines)).
func (m *M2Comprehension) SelectSamples(targets []*types.AnalysisTarget) []Sample {
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
			if file.Lines < 30 { // Skip very small files
				continue
			}

			content := string(file.Content)

			// Count complexity indicators
			complexityCount := 0
			for _, pattern := range complexityPatterns {
				matches := pattern.FindAllString(content, -1)
				complexityCount += len(matches)
			}

			if complexityCount < 5 { // Skip simple files
				continue
			}

			// Score = complexity / sqrt(lines) - favors dense complexity
			score := float64(complexityCount) / math.Sqrt(float64(file.Lines))

			candidates = append(candidates, Sample{
				FilePath:       file.Path,
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
func (m *M2Comprehension) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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
			Duration: time.Since(sampleStart),
		}

		if err != nil {
			sr.Error = err.Error()
			sr.Score = 0
		} else {
			// Heuristic scoring based on response quality indicators
			sr.Score = m.scoreComprehensionResponse(response)
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

// scoreComprehensionResponse uses heuristics to score the comprehension explanation.
func (m *M2Comprehension) scoreComprehensionResponse(response string) int {
	response = strings.ToLower(response)

	score := 5 // Base score

	// Positive indicators (depth of understanding)
	positiveIndicators := []string{
		"returns", "return value", "returns the",
		"error", "handles", "handling",
		"if ", "when ", "condition",
		"loop", "iterate", "for each",
		"edge case", "corner case", "boundary",
		"side effect", "modifies", "updates",
		"validates", "checks", "ensures",
	}

	for _, indicator := range positiveIndicators {
		if strings.Contains(response, indicator) {
			score++
		}
	}

	// Negative indicators (superficial or wrong)
	negativeIndicators := []string{
		"i don't know", "unclear", "cannot determine",
		"might", "probably", "seems to",
		"not sure", "unsure",
	}

	for _, indicator := range negativeIndicators {
		if strings.Contains(response, indicator) {
			score--
		}
	}

	// Length bonus (detailed explanations are better, up to a point)
	wordCount := len(strings.Fields(response))
	if wordCount > 100 {
		score++
	}
	if wordCount > 200 {
		score++
	}

	// Clamp to 1-10
	if score < 1 {
		score = 1
	}
	if score > 10 {
		score = 10
	}

	return score
}
