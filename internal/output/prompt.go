package output

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/ingo-eichhorst/agent-readyness/internal/scoring"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for prompt generation.
const (
	targetScoreOffset = 2.0
	maxScorePrompt    = 10.0
	maxEvidenceItems  = 5
)

// promptParams holds all data needed to render an improvement prompt.
type promptParams struct {
	CategoryName    string // e.g., "C1"
	CategoryDisplay string // e.g., "C1: Code Health"
	CategoryImpact  string // from categoryImpact()
	MetricName      string // e.g., "complexity_avg"
	MetricDisplay   string // e.g., "Complexity avg"
	RawValue        float64
	FormattedValue  string
	Score           float64
	TargetScore     float64 // computed via nextTarget
	TargetValue     float64 // computed via nextTarget
	HasBreakpoints  bool    // false for C7
	Evidence        []types.EvidenceItem
	Language        string // detected language for build commands
}

// renderImprovementPrompt builds HTML containing a copyable prompt with
// 4 sections: Context, Build & Test, Task, and Verification.
func renderImprovementPrompt(params promptParams) string {
	var b strings.Builder

	// Build the plain-text prompt content
	var prompt strings.Builder

	// Section 1: Context
	prompt.WriteString("## Context\n\n")
	prompt.WriteString(fmt.Sprintf("I'm working on improving the %s metric in this codebase.\n", params.MetricDisplay))
	prompt.WriteString(fmt.Sprintf("Current score: %.1f/10 (raw value: %s)\n", params.Score, params.FormattedValue))
	if params.HasBreakpoints {
		prompt.WriteString(fmt.Sprintf("Target score: %.1f/10 (target value: %.4g)\n", params.TargetScore, params.TargetValue))
	} else {
		targetScore := params.Score + targetScoreOffset
		if targetScore > maxScorePrompt {
			targetScore = maxScorePrompt
		}
		prompt.WriteString(fmt.Sprintf("Target score: Improve score above %.1f/10\n", targetScore))
	}
	prompt.WriteString(fmt.Sprintf("\nCategory: %s\n", params.CategoryDisplay))
	prompt.WriteString(fmt.Sprintf("Why it matters: %s\n", params.CategoryImpact))

	// Section 2: Build & Test Commands
	prompt.WriteString("\n## Build & Test Commands\n\n")
	prompt.WriteString(languageBuildCommands(params.Language))
	prompt.WriteString("\n")

	// Section 3: Task
	prompt.WriteString("\n## Task\n\n")
	prompt.WriteString(getMetricTaskGuidance(params.MetricName, params.RawValue, params.TargetValue, params.HasBreakpoints))

	// Files to Focus On (only if evidence exists)
	if len(params.Evidence) > 0 {
		prompt.WriteString("\n### Files to Focus On\n\n")
		limit := len(params.Evidence)
		if limit > maxEvidenceItems {
			limit = maxEvidenceItems
		}
		for i := 0; i < limit; i++ {
			ev := params.Evidence[i]
			prompt.WriteString(fmt.Sprintf("%d. %s:%d - %s (value: %.4g)\n", i+1, ev.FilePath, ev.Line, ev.Description, ev.Value))
		}
	}

	// Section 4: Verification
	prompt.WriteString("\n## Verification\n\n")
	prompt.WriteString("After making changes:\n")
	prompt.WriteString(languageTestCommand(params.Language))
	prompt.WriteString("\nThen re-scan: ars scan . --output-html /tmp/report.html\n")
	prompt.WriteString(fmt.Sprintf("Check that the %s score has improved above %.1f.\n", params.MetricDisplay, params.TargetScore))

	// Wrap in HTML container with copy button
	promptText := prompt.String()
	escapedPrompt := template.HTMLEscapeString(promptText)

	b.WriteString(`<div class="prompt-copy-container">`)
	b.WriteString(`<button class="trace-copy-btn" onclick="copyPromptText(this)">Copy</button>`)
	b.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, escapedPrompt))
	b.WriteString(`</div>`)

	return b.String()
}

// nextTarget computes the next achievable breakpoint from the current score.
// Returns the target raw value and target score.
// Handles both ascending (higher value = higher score) and descending
// (higher value = lower score) breakpoint directions.
func nextTarget(score float64, breakpoints []scoring.Breakpoint) (targetValue float64, targetScore float64) {
	if len(breakpoints) == 0 {
		return 0, score
	}

	// Already at max
	if score >= maxScorePrompt {
		return breakpoints[len(breakpoints)-1].Value, maxScorePrompt
	}

	// Detect direction: ascending means first breakpoint score < last breakpoint score
	ascending := breakpoints[0].Score < breakpoints[len(breakpoints)-1].Score

	if ascending {
		// Higher value = higher score (e.g., coverage: 0%->1, 90%->10)
		// Find the first breakpoint with a score higher than current
		for _, bp := range breakpoints {
			if bp.Score > score {
				return bp.Value, bp.Score
			}
		}
		// Already at or above max; return last breakpoint
		last := breakpoints[len(breakpoints)-1]
		return last.Value, last.Score
	}

	// Descending: higher value = lower score (e.g., complexity: 1->10, 40->1)
	// Breakpoints are sorted by Value ascending, Score descending.
	// Find the last breakpoint (highest value) with score higher than current.
	// We want the closest improvement target, so iterate from high value to low.
	for i := len(breakpoints) - 1; i >= 0; i-- {
		bp := breakpoints[i]
		if bp.Score > score {
			return bp.Value, bp.Score
		}
	}
	// Already at or above max; return first breakpoint (lowest value = highest score)
	first := breakpoints[0]
	return first.Value, first.Score
}

// languageBuildCommands returns build/test commands for the detected language.
func languageBuildCommands(lang string) string {
	var cmds string
	switch strings.ToLower(lang) {
	case "go":
		cmds = "go build ./...\ngo test ./..."
	case "python":
		cmds = "python -m pytest"
	case "typescript":
		cmds = "npm test"
	default:
		cmds = "# (adjust build/test commands for your project)"
	}
	return cmds + "\n(adjust commands for your project if different)"
}

// languageTestCommand returns the test-only command for the detected language.
func languageTestCommand(lang string) string {
	switch strings.ToLower(lang) {
	case "go":
		return "go test ./..."
	case "python":
		return "python -m pytest"
	case "typescript":
		return "npm test"
	default:
		return "# run your test suite"
	}
}

// getMetricTaskGuidance extracts improvement guidance from metric descriptions
// and prepends a metric-specific action line.
func getMetricTaskGuidance(metricName string, rawValue float64, targetValue float64, hasBreakpoints bool) string {
	var b strings.Builder

	// Prepend metric-specific action line
	if hasBreakpoints {
		displayName := metricDisplayName(metricName)
		b.WriteString(fmt.Sprintf("Improve the %s from %.4g to %.4g or better.\n\n", displayName, rawValue, targetValue))
	}

	// Extract "How to Improve" bullet points from descriptions
	desc := getMetricDescription(metricName)
	if string(desc.Detailed) != "" {
		bullets := extractHowToImprove(string(desc.Detailed))
		if bullets != "" {
			b.WriteString("Guidance:\n")
			b.WriteString(bullets)
		} else {
			b.WriteString("Review the metric description for improvement guidance.\n")
		}
	} else {
		b.WriteString("Review the metric description for improvement guidance.\n")
	}

	return b.String()
}

// liTagRe matches <li>...</li> content (non-greedy).
var liTagRe = regexp.MustCompile(`<li>(.*?)</li>`)

// htmlTagRe matches any HTML tag for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// extractHowToImprove parses the Detailed HTML description to find the
// "How to Improve" section and extract <li> items as plain text bullet points.
func extractHowToImprove(detailed string) string {
	// Find the "How to Improve" section
	idx := strings.Index(detailed, "<h4>How to Improve</h4>")
	if idx < 0 {
		return ""
	}

	// Extract content after the heading until the next <h4> or end
	rest := detailed[idx+len("<h4>How to Improve</h4>"):]
	nextH4 := strings.Index(rest, "<h4>")
	if nextH4 >= 0 {
		rest = rest[:nextH4]
	}

	// Extract <li> items
	matches := liTagRe.FindAllStringSubmatch(rest, -1)
	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	for _, match := range matches {
		// Strip any remaining HTML tags from the li content
		text := htmlTagRe.ReplaceAllString(match[1], "")
		text = strings.TrimSpace(text)
		if text != "" {
			b.WriteString(fmt.Sprintf("- %s\n", text))
		}
	}

	return b.String()
}
