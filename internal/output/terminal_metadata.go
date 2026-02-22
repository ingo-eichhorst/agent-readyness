package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// RenderRepoMetadata prints repository metadata to w.
func RenderRepoMetadata(w io.Writer, meta *types.RepoMetadata) {
	bold := color.New(color.Bold)

	fmt.Fprintln(w)
	bold.Fprintln(w, "Repository Metadata")
	fmt.Fprintln(w, "────────────────────────────────────────")

	fmt.Fprintf(w, "  Lines of code:       %d\n", meta.LinesOfCode)

	if meta.Available {
		fmt.Fprintf(w, "  Total commits:       %d\n", meta.TotalCommits)
		fmt.Fprintf(w, "  Contributors:        %d\n", meta.ContributorCount)

		if !meta.FirstCommitDate.IsZero() {
			fmt.Fprintf(w, "  First commit:        %s\n", meta.FirstCommitDate.Format("2006-01-02"))
		}
		if !meta.LastCommitDate.IsZero() {
			fmt.Fprintf(w, "  Last commit:         %s\n", meta.LastCommitDate.Format("2006-01-02"))
		}
	}

	if len(meta.LanguageBreakdown) > 0 {
		fmt.Fprintf(w, "  Languages:           %s\n", formatLanguageBreakdown(meta.LanguageBreakdown))
	}
}

// formatLanguageBreakdown formats language percentages as "Go 85.2%, Python 14.8%".
func formatLanguageBreakdown(breakdown map[types.Language]float64) string {
	type langPct struct {
		lang types.Language
		pct  float64
	}
	sorted := make([]langPct, 0, len(breakdown))
	for lang, pct := range breakdown {
		sorted = append(sorted, langPct{lang, pct})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].pct > sorted[j].pct
	})

	parts := make([]string, len(sorted))
	for i, lp := range sorted {
		name := string(lp.lang)
		if len(name) > 0 {
			name = strings.ToUpper(name[:1]) + name[1:]
		}
		parts[i] = fmt.Sprintf("%s %.1f%%", name, lp.pct)
	}
	return strings.Join(parts, ", ")
}
