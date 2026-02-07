package scoring

import (
	"fmt"

	"github.com/ingo/agent-readyness/pkg/types"
)

// buildTopNEvidence creates evidence items from the first N elements of a pre-sorted slice.
// The descFn generates the description string for each element.
func buildTopNEvidence[T any](
	items []T,
	limit int,
	filePath func(T) string,
	line func(T) int,
	value func(T) float64,
	descFn func(T) string,
) []types.EvidenceItem {
	if len(items) == 0 {
		return []types.EvidenceItem{}
	}

	if len(items) < limit {
		limit = len(items)
	}

	evidence := make([]types.EvidenceItem, limit)
	for i := 0; i < limit; i++ {
		evidence[i] = types.EvidenceItem{
			FilePath:    filePath(items[i]),
			Line:        line(items[i]),
			Value:       value(items[i]),
			Description: descFn(items[i]),
		}
	}
	return evidence
}

// buildFunctionComplexityEvidence creates evidence for top complex functions.
func buildFunctionComplexityEvidence(functions []types.FunctionMetric, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		functions,
		limit,
		func(f types.FunctionMetric) string { return f.File },
		func(f types.FunctionMetric) int { return f.Line },
		func(f types.FunctionMetric) float64 { return float64(f.Complexity) },
		func(f types.FunctionMetric) string {
			return fmt.Sprintf("%s has complexity %d", f.Name, f.Complexity)
		},
	)
}

// buildFunctionLengthEvidence creates evidence for top long functions.
func buildFunctionLengthEvidence(functions []types.FunctionMetric, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		functions,
		limit,
		func(f types.FunctionMetric) string { return f.File },
		func(f types.FunctionMetric) int { return f.Line },
		func(f types.FunctionMetric) float64 { return float64(f.LineCount) },
		func(f types.FunctionMetric) string {
			return fmt.Sprintf("%s is %d lines", f.Name, f.LineCount)
		},
	)
}

// pkgCount is a helper type for package coupling evidence.
type pkgCount struct {
	pkg   string
	count int
}

// buildPackageCouplingEvidence creates evidence for top coupled packages.
func buildPackageCouplingEvidence(entries []pkgCount, limit int, direction string) []types.EvidenceItem {
	return buildTopNEvidence(
		entries,
		limit,
		func(e pkgCount) string { return e.pkg },
		func(e pkgCount) int { return 0 },
		func(e pkgCount) float64 { return float64(e.count) },
		func(e pkgCount) string {
			if direction == "afferent" {
				return fmt.Sprintf("imported by %d packages", e.count)
			}
			return fmt.Sprintf("imports %d packages", e.count)
		},
	)
}

// buildDuplicateBlockEvidence creates evidence for duplicate code blocks.
func buildDuplicateBlockEvidence(blocks []types.DuplicateBlock, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		blocks,
		limit,
		func(b types.DuplicateBlock) string { return b.FileA },
		func(b types.DuplicateBlock) int { return b.StartA },
		func(b types.DuplicateBlock) float64 { return float64(b.LineCount) },
		func(b types.DuplicateBlock) string {
			return fmt.Sprintf("%d-line duplicate block", b.LineCount)
		},
	)
}

// buildTestDependencyEvidence creates evidence for tests with external dependencies.
func buildTestDependencyEvidence(tests []types.TestFunctionMetric, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		tests,
		limit,
		func(t types.TestFunctionMetric) string { return t.File },
		func(t types.TestFunctionMetric) int { return t.Line },
		func(t types.TestFunctionMetric) float64 { return 1 },
		func(t types.TestFunctionMetric) string {
			return fmt.Sprintf("%s has external dependency", t.Name)
		},
	)
}

// buildTestAssertionEvidence creates evidence for tests with low assertion counts.
func buildTestAssertionEvidence(tests []types.TestFunctionMetric, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		tests,
		limit,
		func(t types.TestFunctionMetric) string { return t.File },
		func(t types.TestFunctionMetric) int { return t.Line },
		func(t types.TestFunctionMetric) float64 { return float64(t.AssertionCount) },
		func(t types.TestFunctionMetric) string {
			return fmt.Sprintf("%s has %d assertions", t.Name, t.AssertionCount)
		},
	)
}

// buildChurnEvidence creates evidence for high-churn files.
func buildChurnEvidence(hotspots []types.FileChurn, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		hotspots,
		limit,
		func(h types.FileChurn) string { return h.Path },
		func(h types.FileChurn) int { return 0 },
		func(h types.FileChurn) float64 { return float64(h.CommitCount) },
		func(h types.FileChurn) string {
			return fmt.Sprintf("%d commits", h.CommitCount)
		},
	)
}

// buildTemporalCouplingEvidence creates evidence for temporally coupled file pairs.
func buildTemporalCouplingEvidence(pairs []types.CoupledPair, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		pairs,
		limit,
		func(p types.CoupledPair) string { return p.FileA },
		func(p types.CoupledPair) int { return 0 },
		func(p types.CoupledPair) float64 { return p.Coupling },
		func(p types.CoupledPair) string {
			return fmt.Sprintf("coupled with %s (%.0f%%)", p.FileB, p.Coupling)
		},
	)
}

// buildAuthorFragmentationEvidence creates evidence for files with many authors.
func buildAuthorFragmentationEvidence(hotspots []types.FileChurn, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		hotspots,
		limit,
		func(h types.FileChurn) string { return h.Path },
		func(h types.FileChurn) int { return 0 },
		func(h types.FileChurn) float64 { return float64(h.AuthorCount) },
		func(h types.FileChurn) string {
			return fmt.Sprintf("%d distinct authors", h.AuthorCount)
		},
	)
}

// buildHotspotConcentrationEvidence creates evidence for hotspot files with many changes.
func buildHotspotConcentrationEvidence(hotspots []types.FileChurn, limit int) []types.EvidenceItem {
	return buildTopNEvidence(
		hotspots,
		limit,
		func(h types.FileChurn) string { return h.Path },
		func(h types.FileChurn) int { return 0 },
		func(h types.FileChurn) float64 { return float64(h.TotalChanges) },
		func(h types.FileChurn) string {
			return fmt.Sprintf("hotspot: %d changes", h.TotalChanges)
		},
	)
}
