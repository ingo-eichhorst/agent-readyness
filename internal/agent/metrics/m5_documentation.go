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

	lineCommentPattern := regexp.MustCompile(`(?m)^\s*(?://|#)`)
	blockCommentStart := regexp.MustCompile(`/\*`)
	blockCommentEnd := regexp.MustCompile(`\*/`)

	for _, target := range targets {
		for _, file := range target.Files {
			if file.Class != types.ClassSource || file.Lines < m5MinFileLOC {
				continue
			}

			density, commentLines := m5CalculateCommentDensity(&file, lineCommentPattern, blockCommentStart, blockCommentEnd)
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

	return m5SelectTopCandidates(candidates, m.sampleCount)
}

// m5CalculateCommentDensity counts line and block comments to compute density.
func m5CalculateCommentDensity(file *types.SourceFile, linePattern, blockStart, blockEnd *regexp.Regexp) (float64, int) {
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	commentLines := m5CountLineComments(lines, linePattern)
	commentLines += m5CountBlockComments(content, blockStart, blockEnd)

	density := float64(commentLines) / float64(file.Lines)
	return density, commentLines
}

// m5CountLineComments counts single-line comments.
func m5CountLineComments(lines []string, pattern *regexp.Regexp) int {
	count := 0
	for _, line := range lines {
		if pattern.MatchString(line) {
			count++
		}
	}
	return count
}

// m5CountBlockComments estimates block comment lines.
func m5CountBlockComments(content string, blockStart, blockEnd *regexp.Regexp) int {
	blockStarts := len(blockStart.FindAllString(content, -1))
	blockEnds := len(blockEnd.FindAllString(content, -1))
	return min(blockStarts, blockEnds) * m5BlockCommentLinesEst
}

// m5SelectTopCandidates sorts and limits candidates by score.
func m5SelectTopCandidates(candidates []Sample, limit int) []Sample {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SelectionScore > candidates[j].SelectionScore
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
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
	return executeStandardMetric(ctx, workDir, samples, executor, executeConfig{
		metricID:   m.ID(),
		metricName: m.Name(),
		timeout:    m.timeout,
		tools:      "Read",
		buildPrompt: func(sample Sample) string {
			return fmt.Sprintf(`Analyze the documentation accuracy in %s.

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
		},
		scoreResponse: m.scoreDocumentationResponse,
	})
}

// scoreDocumentationResponse uses grouped heuristics to score the documentation analysis.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups: each group contributes +1 if ANY member matches.
// This prevents saturation where many overlapping indicators all score individually.
func (m *m5Documentation) scoreDocumentationResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)

	trace := ScoreTrace{BaseScore: m5BaseScore}

	trace.Indicators = append(trace.Indicators, matchGroups(responseLower, []indicatorGroup{
		{"structure_summary", []string{"## summary"}},
		{"accurate_section", []string{"accurate documentation", "## accurate"}},
		{"mismatch_section", []string{"potential mismatch", "## potential"}},
		{"specific_analysis", []string{"location:", "comment says", "code does", "issue:"}},
		{"quality_language", []string{"accurate", "correctly", "describes", "matches", "documentation"}},
		{"conclusion", []string{"all documentation appears accurate", "no mismatches found", "documentation is accurate"}},
	})...)

	// Conclusion also matches if both "mismatch" and "line" appear
	if strings.Contains(responseLower, "mismatch") && strings.Contains(responseLower, "line") {
		// Find the conclusion indicator and mark it matched if not already
		for i := range trace.Indicators {
			if trace.Indicators[i].Name == "group:conclusion" && !trace.Indicators[i].Matched {
				trace.Indicators[i].Matched = true
				trace.Indicators[i].Delta = 1
				break
			}
		}
	}

	trace.Indicators = append(trace.Indicators, matchNegativeIndicators(responseLower, []string{
		"cannot analyze", "unable to", "error reading",
		"no comments", "file not found",
	})...)

	return computeScore(&trace), trace
}
