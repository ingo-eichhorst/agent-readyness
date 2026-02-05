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

// M5Documentation measures the agent's ability to detect comment/code mismatches.
// It tests the agent's understanding of documentation accuracy.
//
// Research basis: Code comment inconsistency detection research shows this is
// a distinct, measurable capability (TSE 2024).
type M5Documentation struct {
	sampleCount int
	timeout     time.Duration
}

// NewM5DocumentationMetric creates a Documentation Accuracy Detection metric.
func NewM5DocumentationMetric() *M5Documentation {
	return &M5Documentation{
		sampleCount: 3,
		timeout:     60 * time.Second,
	}
}

func (m *M5Documentation) ID() string { return "documentation_accuracy_detection" }
func (m *M5Documentation) Name() string { return "Documentation Accuracy Detection" }
func (m *M5Documentation) Description() string {
	return "Measures ability to detect comment/code mismatches"
}
func (m *M5Documentation) Timeout() time.Duration { return m.timeout }
func (m *M5Documentation) SampleCount() int { return m.sampleCount }

// SelectSamples picks files with high comment density (> 5% comment lines).
// Higher comment density = more opportunity to detect mismatches.
func (m *M5Documentation) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []Sample

	// Pattern for comment lines (language-agnostic basics)
	lineCommentPattern := regexp.MustCompile(`(?m)^\s*(?://|#)`)
	blockCommentStart := regexp.MustCompile(`/\*`)
	blockCommentEnd := regexp.MustCompile(`\*/`)

	for _, target := range targets {
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}
			if file.Lines < 20 { // Skip very small files
				continue
			}

			content := string(file.Content)
			lines := strings.Split(content, "\n")

			// Count comment lines
			commentLines := 0

			// Count line comments
			for _, line := range lines {
				if lineCommentPattern.MatchString(line) {
					commentLines++
				}
			}

			// Count block comment lines (rough estimation)
			blockStarts := len(blockCommentStart.FindAllString(content, -1))
			blockEnds := len(blockCommentEnd.FindAllString(content, -1))
			// Estimate average block comment is 3 lines
			blockLines := min(blockStarts, blockEnds) * 3
			commentLines += blockLines

			// Calculate comment density
			density := float64(commentLines) / float64(file.Lines)

			// Skip files with very low comment density
			if density < 0.05 {
				continue
			}

			candidates = append(candidates, Sample{
				FilePath:       file.Path,
				SelectionScore: density,
				Description:    fmt.Sprintf("Comment density %.1f%% (%d comment lines)", density*100, commentLines),
			})
		}
	}

	// Sort by density descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SelectionScore > candidates[j].SelectionScore
	})

	if len(candidates) > m.sampleCount {
		candidates = candidates[:m.sampleCount]
	}
	return candidates
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// documentationRubric is the LLM-as-judge rubric for scoring documentation accuracy detection.
const documentationRubric = `You are evaluating an AI coding agent's ability to detect documentation issues.

The agent was asked to identify mismatches between code comments and actual code behavior.

Score the response from 1-10 based on these criteria:
- Detection accuracy (60%): Did the agent correctly identify mismatches (or confirm accuracy)?
- Explanation quality (40%): Are the identified issues clearly explained?

Consider:
- Score 9-10: Identifies all mismatches with correct explanations, or correctly states none exist
- Score 7-8: Identifies most issues with good explanations
- Score 4-6: Identifies obvious issues only, some false positives/negatives
- Score 1-3: Cannot reliably detect mismatches, many errors

Respond with JSON only: {"score": N, "reason": "brief explanation"}`

// Execute asks the agent to detect documentation accuracy issues.
func (m *M5Documentation) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

		prompt := fmt.Sprintf(`Analyze the documentation accuracy in %s.

Your task:
1. Read the file and identify all comments (line comments, block comments, doc strings)
2. For each comment, check if it accurately describes the adjacent code
3. Report any mismatches where comments don't match code behavior

Format your response as:
## Summary
[Overall documentation accuracy: good/moderate/poor]

## Accurate Documentation
[List comments that correctly describe the code]

## Potential Mismatches
[List any comments that may be outdated, incorrect, or misleading]
For each mismatch:
- Location: [line number or code reference]
- Comment says: [what the comment claims]
- Code does: [what the code actually does]
- Issue: [why this is a mismatch]

If all documentation appears accurate, state that clearly.`, sample.FilePath)

		response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read", timePerSample)
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
			sr.Score = m.scoreDocumentationResponse(response)
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

// scoreDocumentationResponse uses heuristics to score the documentation analysis.
func (m *M5Documentation) scoreDocumentationResponse(response string) int {
	responseLower := strings.ToLower(response)

	score := 5 // Base score

	// Check for structured response (indicates thorough analysis)
	if strings.Contains(responseLower, "## summary") {
		score++
	}
	if strings.Contains(responseLower, "accurate documentation") || strings.Contains(responseLower, "## accurate") {
		score++
	}
	if strings.Contains(responseLower, "potential mismatch") || strings.Contains(responseLower, "## potential") {
		score++
	}

	// Positive indicators (detailed analysis)
	positiveIndicators := []struct {
		pattern string
		weight  int
	}{
		{"location:", 1},
		{"line", 1},
		{"comment says", 1},
		{"code does", 1},
		{"issue:", 1},
		{"accurate", 1},
		{"correctly", 1},
		{"describes", 1},
		{"matches", 1},
		{"documentation", 1},
	}

	for _, ind := range positiveIndicators {
		if strings.Contains(responseLower, ind.pattern) {
			score += ind.weight
		}
	}

	// Check for clear conclusion
	if strings.Contains(responseLower, "all documentation appears accurate") ||
		strings.Contains(responseLower, "no mismatches found") ||
		strings.Contains(responseLower, "documentation is accurate") {
		score++ // Clear positive conclusion
	}

	if strings.Contains(responseLower, "mismatch") && strings.Contains(responseLower, "line") {
		score++ // Found and located mismatches
	}

	// Negative indicators
	negativeIndicators := []string{
		"cannot analyze", "unable to", "error reading",
		"no comments", "file not found",
	}

	for _, indicator := range negativeIndicators {
		if strings.Contains(responseLower, indicator) {
			score--
		}
	}

	// Length check (thorough analysis should be detailed)
	wordCount := len(strings.Fields(response))
	if wordCount < 50 {
		score--
	}
	if wordCount > 100 {
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
