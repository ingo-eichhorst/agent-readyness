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

// M3Navigation measures the agent's ability to trace dependencies across files.
// It tests cross-file understanding and data flow tracing.
//
// Research basis: RepoGraph (ICLR 2025) shows 32.8% improvement when agents
// have repository-level understanding.
type M3Navigation struct {
	sampleCount int
	timeout     time.Duration
}

// NewM3NavigationMetric creates a Cross-File Navigation metric.
func NewM3NavigationMetric() *M3Navigation {
	return &M3Navigation{
		sampleCount: 2,
		timeout:     120 * time.Second,
	}
}

func (m *M3Navigation) ID() string { return "cross_file_navigation" }
func (m *M3Navigation) Name() string { return "Cross-File Navigation" }
func (m *M3Navigation) Description() string {
	return "Measures ability to trace dependencies across files"
}
func (m *M3Navigation) Timeout() time.Duration { return m.timeout }
func (m *M3Navigation) SampleCount() int { return m.sampleCount }

// SelectSamples picks files with many imports (dependency entry points).
// Sorts by import count descending, takes top 2 non-test files.
func (m *M3Navigation) SelectSamples(targets []*types.AnalysisTarget) []Sample {
	var candidates []Sample

	// Import patterns for different languages
	importPatterns := map[types.Language]*regexp.Regexp{
		types.LangGo:         regexp.MustCompile(`(?m)^\s*(?:import\s+|"[^"]+"\s*$|\t"[^"]+")`),
		types.LangPython:     regexp.MustCompile(`(?m)^(?:import|from)\s+\w+`),
		types.LangTypeScript: regexp.MustCompile(`(?m)^import\s+.+from\s+['"]`),
	}

	for _, target := range targets {
		pattern, ok := importPatterns[target.Language]
		if !ok {
			// Default pattern for unknown languages
			pattern = regexp.MustCompile(`(?m)^(?:import|from|require|include)\s+`)
		}

		for _, file := range target.Files {
			if file.Class != types.ClassSource {
				continue
			}

			content := string(file.Content)
			matches := pattern.FindAllString(content, -1)
			importCount := len(matches)

			if importCount < 3 { // Skip files with few imports
				continue
			}

			candidates = append(candidates, Sample{
				FilePath:       file.Path,
				SelectionScore: float64(importCount),
				Description:    fmt.Sprintf("High import count (%d imports)", importCount),
			})
		}
	}

	// Sort by import count descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].SelectionScore > candidates[j].SelectionScore
	})

	if len(candidates) > m.sampleCount {
		candidates = candidates[:m.sampleCount]
	}
	return candidates
}

// navigationRubric is the LLM-as-judge rubric for scoring navigation.
const navigationRubric = `You are evaluating an AI coding agent's ability to trace code across files.

The agent was asked to trace a dependency or data flow across files.

Score the response from 1-10 based on these criteria:
- Completeness (50%): Did the agent identify all relevant files in the chain?
- Accuracy (30%): Are the file paths, functions, and relationships correct?
- Clarity (20%): Is the trace clearly presented and easy to follow?

Consider:
- Score 9-10: Complete trace with all files/functions correctly identified
- Score 7-8: Most files found, minor gaps in the chain
- Score 4-6: Only direct dependencies, missing deeper connections
- Score 1-3: Cannot navigate beyond single file

Respond with JSON only: {"score": N, "reason": "brief explanation"}`

// Execute asks the agent to trace dependencies for each sample.
func (m *M3Navigation) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
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

		prompt := fmt.Sprintf(`Examine the file at %s and trace its dependencies.

Your task:
1. List all imports/dependencies in this file
2. For each imported module/package, identify what it provides
3. Trace the data flow: pick one function and show how data flows from this file through other files

Format your response as:
- Imports: [list of imports]
- Dependency Purpose: [for each import, what it provides]
- Data Flow Trace: [starting function] -> [calls in other files] -> [final destination]

Reference actual file paths and function names from the codebase.`, sample.FilePath)

		response, err := executor.ExecutePrompt(sampleCtx, workDir, prompt, "Read,Glob,Grep", timePerSample)
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
			sr.Score = m.scoreNavigationResponse(response)
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

// scoreNavigationResponse uses heuristics to score the navigation trace.
func (m *M3Navigation) scoreNavigationResponse(response string) int {
	responseLower := strings.ToLower(response)

	score := 5 // Base score

	// Positive indicators (cross-file understanding)
	positiveIndicators := []struct {
		pattern string
		weight  int
	}{
		{"import", 1},
		{"from", 1},
		{"->", 2}, // Data flow arrows
		{"calls", 1},
		{"returns", 1},
		{".go", 1}, // File references
		{".py", 1},
		{".ts", 1},
		{".js", 1},
		{"package", 1},
		{"module", 1},
		{"function", 1},
		{"exports", 1},
		{"provides", 1},
		{"dependency", 1},
		{"flow", 1},
	}

	for _, ind := range positiveIndicators {
		if strings.Contains(responseLower, ind.pattern) {
			score += ind.weight
		}
	}

	// Count file path references (indicates multi-file navigation)
	pathCount := strings.Count(response, "/")
	if pathCount > 3 {
		score++
	}
	if pathCount > 6 {
		score++
	}

	// Negative indicators
	negativeIndicators := []string{
		"cannot find", "not found", "no file",
		"unable to", "cannot trace", "unknown",
	}

	for _, indicator := range negativeIndicators {
		if strings.Contains(responseLower, indicator) {
			score--
		}
	}

	// Length check (navigation should be detailed)
	wordCount := len(strings.Fields(response))
	if wordCount < 50 {
		score--
	}
	if wordCount > 150 {
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
