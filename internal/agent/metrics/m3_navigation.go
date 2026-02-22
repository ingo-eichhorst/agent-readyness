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

// M3 sample selection and scoring constants.
const (
	m3SampleCount       = 2                // Number of code samples to evaluate
	m3Timeout           = 360 * time.Second // Total timeout across all samples
	m3MinImports        = 3                // Minimum imports for sample selection
	m3BaseScore         = 2                // Starting score before heuristic adjustments
	m3DepthPathCount    = 6                // Min path reference count for depth indicator
	m3ExtensiveWordCount = 200             // Min word count for extensive response indicator
)

// m3Navigation measures the agent's ability to trace dependencies across files.
// It tests cross-file understanding and data flow tracing.
//
// Research basis: RepoGraph (ICLR 2025) shows 32.8% improvement when agents
// have repository-level understanding.
type m3Navigation struct {
	sampleCount int
	timeout     time.Duration
}

// newM3NavigationMetric creates a Cross-File Navigation metric.
func newM3NavigationMetric() *m3Navigation {
	return &m3Navigation{
		sampleCount: m3SampleCount,
		timeout:     m3Timeout,
	}
}

// ID returns the metric identifier.
func (m *m3Navigation) ID() string { return "cross_file_navigation" }

// Name returns the human-readable metric name.
func (m *m3Navigation) Name() string { return "Cross-File Navigation" }

// Description returns what this metric measures.
func (m *m3Navigation) Description() string {
	return "Measures ability to trace dependencies across files"
}

// Timeout returns the per-metric timeout duration.
func (m *m3Navigation) Timeout() time.Duration { return m.timeout }

// SampleCount returns the number of samples to evaluate.
func (m *m3Navigation) SampleCount() int { return m.sampleCount }

// SelectSamples picks files with many imports (dependency entry points).
// Sorts by import count descending, takes top 2 non-test files.
func (m *m3Navigation) SelectSamples(targets []*types.AnalysisTarget) []Sample {
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

			if importCount < m3MinImports { // Skip files with few imports
				continue
			}

			candidates = append(candidates, Sample{
				FilePath:       file.RelPath,
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
func (m *m3Navigation) Execute(ctx context.Context, workDir string, samples []Sample, executor Executor) MetricResult {
	return executeStandardMetric(ctx, workDir, samples, executor, executeConfig{
		metricID:   m.ID(),
		metricName: m.Name(),
		timeout:    m.timeout,
		tools:      "Read,Glob,Grep",
		buildPrompt: func(sample Sample) string {
			return fmt.Sprintf(`Examine the file at %s and trace its dependencies.

Your task:
1. List all imports/dependencies in this file
2. For each imported module/package, identify what it provides
3. Trace the data flow: pick one function and show how data flows from this file through other files

Format your response as:
- Imports: [list of imports]
- Dependency Purpose: [for each import, what it provides]
- Data Flow Trace: [starting function] -> [calls in other files] -> [final destination]

Reference actual file paths and function names from the codebase.`, sample.FilePath)
		},
		scoreResponse: m.scoreNavigationResponse,
	})
}

// scoreNavigationResponse uses grouped heuristics to score the navigation trace.
// The ScoreTrace is the source of truth: FinalScore = BaseScore + sum(Deltas), clamped.
//
// Scoring uses thematic groups: each group contributes +1 if ANY member matches.
// This prevents saturation where many overlapping indicators all score individually.
func (m *m3Navigation) scoreNavigationResponse(response string) (int, ScoreTrace) {
	responseLower := strings.ToLower(response)

	trace := ScoreTrace{BaseScore: m3BaseScore}

	trace.Indicators = append(trace.Indicators, matchGroups(responseLower, []indicatorGroup{
		{"import_awareness", []string{"import", "from"}},
		{"cross_file_refs", []string{".go", ".py", ".ts", ".js"}},
		{"data_flow", []string{"->", "flow"}},
		{"purpose_mapping", []string{"module", "provides", "exports", "purpose"}},
	})...)

	// Depth group: based on file path reference count
	pathCount := strings.Count(response, "/")
	matchedDepth := pathCount > m3DepthPathCount
	deltaDepth := 0
	if matchedDepth {
		deltaDepth = 1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "group:depth", Matched: matchedDepth, Delta: deltaDepth,
	})

	// Extensive depth group: lengthy response with many paths
	wordCount := len(strings.Fields(response))
	matchedExtensive := wordCount > m3ExtensiveWordCount
	deltaExtensive := 0
	if matchedExtensive {
		deltaExtensive = 1
	}
	trace.Indicators = append(trace.Indicators, IndicatorMatch{
		Name: "group:extensive_depth", Matched: matchedExtensive, Delta: deltaExtensive,
	})

	trace.Indicators = append(trace.Indicators, matchNegativeIndicators(responseLower, []string{
		"cannot find", "not found", "no file",
		"unable to", "cannot trace", "unknown",
	})...)

	return computeScore(&trace), trace
}
