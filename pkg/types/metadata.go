package types

import "time"

// RepoMetadata holds high-level repository metadata for quick understanding
// of a repo's state and maturity. These are informational metrics (not scored).
type RepoMetadata struct {
	Available         bool                // false if not a git repo
	LinesOfCode       int                 // total source LOC (excluding tests, generated, vendor)
	FirstCommitDate   time.Time           // date of the first commit
	LastCommitDate    time.Time           // date of the most recent commit
	ContributorCount  int                 // number of unique contributors
	TotalCommits      int                 // total number of commits
	LanguageBreakdown map[Language]float64 // language -> percentage of source LOC
}
