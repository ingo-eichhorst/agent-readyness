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

// M1 sample selection and scoring constants.
const (
	m1SampleCount    = 1               // One file, run multiple times
	m1Timeout        = 540 * time.Second // 3 runs of 180s each
	m1Runs           = 3               // Number of repeat runs per sample
	m1MinFileLOC     = 50              // Minimum file size for sample selection
	m1MaxFileLOC     = 200             // Maximum file size for sample selection
	m1MinFuncCount   = 3               // Minimum functions for sample selection
	m1MaxFuncCount   = 10              // Maximum functions for sample selection
	m1IdealFileLOC   = 100             // Ideal file size for scoring
	m1IdealFuncCount = 5               // Ideal function count for scoring
	m1FallbackMinLOC = 20              // Minimum lines for fallback selection
	m1VarianceNorm   = 100.0           // Variance normalization factor
	m1LowVariance    = 5.0             // Variance % below which score is excellent
	m1MedVariance    = 15.0            // Variance % below which score is good
	m1HighVariance   = 30.0            // Variance % below which score is fair
)

// M1 score trace delta values.
const (
	m1DeltaExactJSON   = 10 // Delta for exact JSON array response
	m1DeltaPartialJSON = 7  // Delta for partial JSON array response
	m1DeltaNonEmpty    = 4  // Delta for non-empty, non-JSON response
	m1DeltaEmpty       = 1  // Delta for empty response
)

// M1Consistency measures task execution reproducibility across multiple runs.
// It tests the same simple task 3 times and measures variance in completion.
//
// Research basis: Agent benchmarks show ~13% variance in results; consistency
// is critical for reliability in production use.
type m1Consistency struct {
	sampleCount int
	timeout     time.Duration
	runs        int // Number of times to repeat the task
}

// newM1ConsistencyMetric creates a Task Execution Consistency metric.
func newM1ConsistencyMetric() *m1Consistency {
	return &m1Consistency{
		sampleCount: m1SampleCount,
		timeout:     m1Timeout,
		runs:        m1Runs,
	}
}

// ID returns the metric identifier.
func (m *m1Consistency) ID() string { return "task_execution_consistency" }

// Name returns the human-readable metric name.
func (m *m1Consistency) Name() string { return "Task Execution Consistency" }

// Description returns what this metric measures.
func (m *m1Consistency) Description() string {
	return "Measures reproducibility of agent task completion across multiple runs"
}

// Timeout returns the per-metric timeout duration.
func (m *m1Consistency) Timeout() time.Duration { return m.timeout }

// SampleCount returns the number of samples to evaluate.
func (m *m1Consistency) SampleCount() int { return m.sampleCount }

// SelectSamples picks 1 file with moderate size (50-200 LOC) and 3-10 functions.
// Uses deterministic heuristics: count `func ` occurrences, prefer moderate complexity.
func (m *m1Consistency) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []Sample

	funcPattern := regexp.MustCompile(`(?m)^func\s+`)

	for _, target := range targets {
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}

			// Skip files outside ideal size range
			if file.Lines < m1MinFileLOC || file.Lines > m1MaxFileLOC {
				continue
			}

			content := string(file.Content)
			funcMatches := funcPattern.FindAllString(content, -1)
			funcCount := len(funcMatches)

			// Prefer files with moderate function count
			if funcCount < m1MinFuncCount || funcCount > m1MaxFuncCount {
				continue
			}

			// Score: prefer files closer to middle of range
			sizeScore := 1.0 - float64(abs(file.Lines-m1IdealFileLOC))/float64(m1IdealFileLOC)
			funcScore := 1.0 - float64(abs(funcCount-m1IdealFuncCount))/float64(m1IdealFuncCount)
			score := (sizeScore + funcScore) / 2

			candidates = append(candidates, Sample{
				FilePath:       file.RelPath,
				SelectionScore: score,
				Description:    fmt.Sprintf("Moderate size (%d LOC, %d funcs)", file.Lines, funcCount),
			})
		}
	}

	if len(candidates) == 0 {
		// Fallback: take any source file
		for _, target := range targets {
			for _, file := range target.Files {
				if file.Class != types.ClassSource && file.Lines > m1FallbackMinLOC {
					candidates = append(candidates, Sample{
						FilePath:       file.RelPath,
						SelectionScore: float64(file.Lines),
						Description:    fmt.Sprintf("Fallback selection (%d LOC)", file.Lines),
					})
				}
			}
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

// Execute runs the same task 3 times and measures variance.
func (m *m1Consistency) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

	sample := samples[0]
	runResults, scores := m.executeRuns(ctx, workDir, sample, executor)

	result.Samples = runResults
	result.Duration = time.Since(startTime)

	if len(scores) == 0 {
		result.Score = 0
		result.Error = "all runs failed"
		return result
	}

	result.Score = m.scoreFromVariance(scores)
	return result
}

// executeRuns runs the task multiple times, returning sample results and valid scores.
func (m *m1Consistency) executeRuns(ctx context.Context, workDir string, sample Sample, executor Executor) ([]SampleResult, []int) {
	var runResults []SampleResult
	scores := make([]int, 0, m.runs)
	perRunTimeout := m.timeout / time.Duration(m.runs)

	for i := 0; i < m.runs; i++ {
		runCtx, cancel := context.WithTimeout(ctx, perRunTimeout)
		prompt := fmt.Sprintf(`Read the file at %s and list all function names defined in it.
Return ONLY a JSON array of function names, e.g.: ["func1", "func2"]
Do not include any explanation, just the JSON array.`, sample.FilePath)

		response, err := executor.ExecutePrompt(runCtx, workDir, prompt, "Read", perRunTimeout)
		cancel()

		sr := SampleResult{Sample: sample, Response: response, Prompt: prompt}
		if err != nil {
			sr.Error = err.Error()
		} else {
			sr.Score, sr.ScoreTrace = scoreM1Response(strings.TrimSpace(response))
			scores = append(scores, sr.Score)
		}
		runResults = append(runResults, sr)
	}
	return runResults, scores
}

// scoreM1Response scores a single run response based on JSON array format.
func scoreM1Response(response string) (int, ScoreTrace) {
	trace := ScoreTrace{BaseScore: 0}

	switch {
	case strings.HasPrefix(response, "[") && strings.HasSuffix(response, "]"):
		trace.Indicators = append(trace.Indicators, IndicatorMatch{
			Name: "json_array_exact", Matched: true, Delta: m1DeltaExactJSON,
		})
	case strings.Contains(response, "["):
		trace.Indicators = append(trace.Indicators,
			IndicatorMatch{Name: "json_array_exact", Matched: false, Delta: 0},
			IndicatorMatch{Name: "json_array_partial", Matched: true, Delta: m1DeltaPartialJSON},
		)
	case len(response) > 0:
		trace.Indicators = append(trace.Indicators,
			IndicatorMatch{Name: "json_array_exact", Matched: false, Delta: 0},
			IndicatorMatch{Name: "json_array_partial", Matched: false, Delta: 0},
			IndicatorMatch{Name: "non_empty_response", Matched: true, Delta: m1DeltaNonEmpty},
		)
	default:
		trace.Indicators = append(trace.Indicators,
			IndicatorMatch{Name: "json_array_exact", Matched: false, Delta: 0},
			IndicatorMatch{Name: "json_array_partial", Matched: false, Delta: 0},
			IndicatorMatch{Name: "non_empty_response", Matched: false, Delta: 0},
			IndicatorMatch{Name: "empty_response", Matched: true, Delta: m1DeltaEmpty},
		)
	}
	return computeScore(&trace), trace
}

// scoreFromVariance converts score variance into a final consistency score.
func (m *m1Consistency) scoreFromVariance(scores []int) int {
	variance := calculateVariance(scores)
	variancePct := (variance / m1VarianceNorm) * m1VarianceNorm

	switch {
	case variancePct < m1LowVariance:
		return maxScore
	case variancePct < m1MedVariance:
		return 7
	case variancePct < m1HighVariance:
		return 4
	default:
		return minScore
	}
}

// calculateVariance computes the variance of a slice of integers.
func calculateVariance(scores []int) float64 {
	if len(scores) == 0 {
		return 0
	}

	// Calculate mean
	sum := 0
	for _, s := range scores {
		sum += s
	}
	mean := float64(sum) / float64(len(scores))

	// Calculate variance
	var sumSquares float64
	for _, s := range scores {
		diff := float64(s) - mean
		sumSquares += diff * diff
	}

	return sumSquares / float64(len(scores))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
