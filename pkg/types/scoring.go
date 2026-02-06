package types

// ScoredResult holds the complete scoring output for a project.
type ScoredResult struct {
	ProjectName string          // Name of the scanned project (basename of root dir)
	Categories  []CategoryScore // Per-category scores (C1, C3, C6)
	Composite   float64         // Weighted composite score (1-10)
	Tier        string          // Tier classification (e.g., "Agent-Ready")
}

// CategoryScore holds the score for one category (e.g., C1 Code Health).
type CategoryScore struct {
	Name      string     // Category identifier (e.g., "C1")
	Score     float64    // Weighted average of sub-scores (1-10)
	Weight    float64    // Weight in composite score
	SubScores []SubScore // Per-metric sub-scores
}

// EvidenceItem represents a single worst-offender for a metric.
type EvidenceItem struct {
	FilePath    string  `json:"file_path"`
	Line        int     `json:"line"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

// SubScore holds the score for a single metric within a category.
type SubScore struct {
	MetricName string         `json:"metric_name"`
	RawValue   float64        `json:"raw_value"`
	Score      float64        `json:"score"`
	Weight     float64        `json:"weight"`
	Available  bool           `json:"available"`
	Evidence   []EvidenceItem `json:"evidence"`
}

// ExitError is returned when the CLI should exit with a specific code.
// For example, threshold violations exit with code 2.
type ExitError struct {
	Code    int
	Message string
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	return e.Message
}
