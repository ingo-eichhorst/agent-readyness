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

// M1Consistency measures task execution reproducibility across multiple runs.
// It tests the same simple task 3 times and measures variance in completion.
//
// Research basis: Agent benchmarks show ~13% variance in results; consistency
// is critical for reliability in production use.
type M1Consistency struct {
	sampleCount int
	timeout     time.Duration
	runs        int // Number of times to repeat the task
}

// NewM1ConsistencyMetric creates a Task Execution Consistency metric.
func NewM1ConsistencyMetric() *M1Consistency {
	return &M1Consistency{
		sampleCount: 1,            // One file, run 3 times
		timeout:     540 * time.Second, // 3 runs of 180s each
		runs:        3,
	}
}

func (m *M1Consistency) ID() string { return "task_execution_consistency" }
func (m *M1Consistency) Name() string { return "Task Execution Consistency" }
func (m *M1Consistency) Description() string {
	return "Measures reproducibility of agent task completion across multiple runs"
}
func (m *M1Consistency) Timeout() time.Duration { return m.timeout }
func (m *M1Consistency) SampleCount() int { return m.sampleCount }

// SelectSamples picks 1 file with moderate size (50-200 LOC) and 3-10 functions.
// Uses deterministic heuristics: count `func ` occurrences, prefer moderate complexity.
func (m *M1Consistency) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []Sample

	funcPattern := regexp.MustCompile(`(?m)^func\s+`)

	for _, target := range targets {
		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}

			// Skip files outside ideal size range
			if file.Lines < 50 || file.Lines > 200 {
				continue
			}

			content := string(file.Content)
			funcMatches := funcPattern.FindAllString(content, -1)
			funcCount := len(funcMatches)

			// Prefer files with 3-10 functions
			if funcCount < 3 || funcCount > 10 {
				continue
			}

			// Score: prefer files closer to middle of range (100 LOC, 5-6 funcs)
			sizeScore := 1.0 - float64(abs(file.Lines-100))/100.0
			funcScore := 1.0 - float64(abs(funcCount-5))/5.0
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
				if file.Class != types.ClassSource && file.Lines > 20 {
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
func (m *M1Consistency) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

	// Run the same task multiple times
	var runResults []SampleResult
	scores := make([]int, 0, m.runs)

	for i := 0; i < m.runs; i++ {
		runCtx, cancel := context.WithTimeout(ctx, m.timeout/time.Duration(m.runs))

		prompt := fmt.Sprintf(`Read the file at %s and list all function names defined in it.
Return ONLY a JSON array of function names, e.g.: ["func1", "func2"]
Do not include any explanation, just the JSON array.`, sample.FilePath)

		response, err := executor.ExecutePrompt(runCtx, workDir, prompt, "Read", m.timeout/time.Duration(m.runs))
		cancel()

		sr := SampleResult{
			Sample:   sample,
			Response: response,
			Prompt:   prompt,
		}

		if err != nil {
			sr.Error = err.Error()
			sr.Score = 0
		} else {
			// Score based on response validity (does it look like a JSON array?)
			response = strings.TrimSpace(response)
			trace := ScoreTrace{BaseScore: 0}

			if strings.HasPrefix(response, "[") && strings.HasSuffix(response, "]") {
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_exact", Matched: true, Delta: 10,
				})
			} else if strings.Contains(response, "[") {
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_exact", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_partial", Matched: true, Delta: 7,
				})
			} else if len(response) > 0 {
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_exact", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_partial", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "non_empty_response", Matched: true, Delta: 4,
				})
			} else {
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_exact", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "json_array_partial", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "non_empty_response", Matched: false, Delta: 0,
				})
				trace.Indicators = append(trace.Indicators, IndicatorMatch{
					Name: "empty_response", Matched: true, Delta: 1,
				})
			}

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
			sr.Score = score
			sr.ScoreTrace = trace
			scores = append(scores, sr.Score)
		}

		runResults = append(runResults, sr)
	}

	result.Samples = runResults
	result.Duration = time.Since(startTime)

	// Calculate variance-based score
	if len(scores) == 0 {
		result.Score = 0
		result.Error = "all runs failed"
		return result
	}

	variance := calculateVariance(scores)
	variancePct := (variance / 100.0) * 100 // Normalize to percentage

	// Score based on variance thresholds from research
	switch {
	case variancePct < 5:
		result.Score = 10
	case variancePct < 15:
		result.Score = 7
	case variancePct < 30:
		result.Score = 4
	default:
		result.Score = 1
	}

	return result
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
