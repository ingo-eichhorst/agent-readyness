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

// M5 sample selection and scoring constants.
const (
	m5SampleCount            = 3                // Number of documentation samples to evaluate
	m5Timeout                = 180 * time.Second // Total timeout across all samples
	m5MinFileLOC             = 20               // Minimum file size for sample selection
	m5MinCommentDensity      = 0.05             // Minimum comment density (5%) for selection
	m5BlockCommentLinesEst   = 3                // Estimated average lines per block comment
	m5PercentMultiplier      = 100              // Multiplier for density-to-percent display
	m5BaseScore              = 3                // Starting score before heuristic adjustments
)

// m5Documentation measures the agent's ability to detect comment/code mismatches.
// It tests the agent's understanding of documentation accuracy.
//
// Research basis: Code comment inconsistency detection research shows this is
// a distinct, measurable capability (TSE 2024).
type m5Documentation struct {
	sampleCount int
	timeout     time.Duration
}

// newM5DocumentationMetric creates a Documentation Accuracy Detection metric.
func newM5DocumentationMetric() *m5Documentation {
	return &m5Documentation{
		sampleCount: m5SampleCount,
		timeout:     m5Timeout,
	}
}

// ID returns the metric identifier.
func (m *m5Documentation) ID() string { return "documentation_accuracy_detection" }

// Name returns the human-readable metric name.
func (m *m5Documentation) Name() string { return "Documentation Accuracy Detection" }

// Description returns what this metric measures.
func (m *m5Documentation) Description() string {
	return "Measures ability to detect comment/code mismatches"
}

// Timeout returns the per-metric timeout duration.
func (m *m5Documentation) Timeout() time.Duration { return m.timeout }

// SampleCount returns the number of samples to evaluate.
func (m *m5Documentation) SampleCount() int { return m.sampleCount }

// SelectSamples picks files with high comment density (> 5% comment lines).
// Higher comment density = more opportunity to detect mismatches.
func (m *m5Documentation) SelectSamples(targets []*types.AnalysisTarget) []Sample {
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
			if file.Lines < m5MinFileLOC { // Skip very small files
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
			blockLines := min(blockStarts, blockEnds) * m5BlockCommentLinesEst
			commentLines += blockLines

			// Calculate comment density
			density := float64(commentLines) / float64(file.Lines)

			// Skip files with very low comment density
			if density < m5MinCommentDensity {
				continue
			}

			candidates = append(candidates, Sample{
				FilePath:       file.RelPath,
				SelectionScore: density,
				Description:    fmt.Sprintf("Comment density %.1f%% (%d comment lines)", density*m5PercentMultiplier, commentLines),
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
func (m *m5Documentation) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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
			Prompt:   prompt,
			Duration: time.Since(sampleStart),
		}

		if err != nil {
			sr.Error = err.Error()
			sr.Score = 0
		} else {
			sr.Score, sr.ScoreTrace = m.scoreDocumentationResponse(response)
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

// scoreDocumentationResponse uses grouped heuristics to score the documentation analysis.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups: each group contributes +1 if ANY member matches.
// This prevents saturation where many overlapping indicators all score individually.
func (m *m5Documentation) scoreDocumentationResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)
	trace := ScoreTrace{BaseScore: m5BaseScore}

	groups := []indicatorGroup{
		{name: "group:structure_summary", patterns: []string{"## summary"}, delta: 1},
		{name: "group:accurate_section", patterns: []string{"accurate documentation", "## accurate"}, delta: 1},
		{name: "group:mismatch_section", patterns: []string{"potential mismatch", "## potential"}, delta: 1},
		{name: "group:specific_analysis", patterns: []string{"location:", "comment says", "code does", "issue:"}, delta: 1},
		{name: "group:quality_language", patterns: []string{"accurate", "correctly", "describes", "matches", "documentation"}, delta: 1},
		{name: "group:conclusion", patterns: []string{"all documentation appears accurate", "no mismatches found", "documentation is accurate"}, delta: 1},
	}
	checkGroups(&trace, responseLower, groups)

	// Conclusion also matches "mismatch" + "line" together
	if strings.Contains(responseLower, "mismatch") && strings.Contains(responseLower, "line") {
		if !trace.Indicators[5].Matched {
			trace.Indicators[5] = IndicatorMatch{Name: "group:conclusion", Matched: true, Delta: 1}
		}
	}

	negatives := []indicatorGroup{
		{name: "negative:cannot analyze", patterns: []string{"cannot analyze"}, delta: -1},
		{name: "negative:unable to", patterns: []string{"unable to"}, delta: -1},
		{name: "negative:error reading", patterns: []string{"error reading"}, delta: -1},
		{name: "negative:no comments", patterns: []string{"no comments"}, delta: -1},
		{name: "negative:file not found", patterns: []string{"file not found"}, delta: -1},
	}
	checkGroups(&trace, responseLower, negatives)

	return computeTraceScore(&trace), trace
}
