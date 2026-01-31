package output

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"github.com/ingo/agent-readyness/pkg/types"
)

// RenderSummary prints a formatted scan summary to w.
// Color is automatically disabled when w is not a TTY (e.g., piped output).
func RenderSummary(w io.Writer, result *types.ScanResult, verbose bool) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Header
	bold.Fprintf(w, "ARS Scan: %s\n", result.RootDir)
	fmt.Fprintln(w, "────────────────────────────────────────")

	// Total count
	fmt.Fprintf(w, "Go files discovered: %d\n", result.TotalFiles)

	// Source and test counts (always shown)
	green.Fprintf(w, "  Source files:        %d\n", result.SourceCount)
	yellow.Fprintf(w, "  Test files:          %d\n", result.TestCount)

	// Excluded categories (only shown if non-zero)
	if result.GeneratedCount > 0 {
		fmt.Fprintf(w, "  Generated (excluded): %d\n", result.GeneratedCount)
	}
	if result.VendorCount > 0 {
		fmt.Fprintf(w, "  Vendor (excluded):    %d\n", result.VendorCount)
	}
	if result.GitignoreCount > 0 {
		fmt.Fprintf(w, "  Gitignored (excluded): %d\n", result.GitignoreCount)
	}

	// Verbose: list individual files
	if verbose {
		fmt.Fprintln(w)
		bold.Fprintln(w, "Discovered files:")
		for _, f := range result.Files {
			tag := f.Class.String()
			suffix := ""
			if f.Class == types.ClassExcluded && f.ExcludeReason != "" {
				suffix = fmt.Sprintf(" (%s)", f.ExcludeReason)
			}
			fmt.Fprintf(w, "  [%s] %s%s\n", tag, f.RelPath, suffix)
		}
	}
}
