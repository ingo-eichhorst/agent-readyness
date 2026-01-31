package types

// ScoredResult holds the complete scoring output for a project.
type ScoredResult struct {
	Categories []CategoryScore // Per-category scores (C1, C3, C6)
	Composite  float64         // Weighted composite score (1-10)
	Tier       string          // Tier classification (e.g., "Agent-Ready")
}

// CategoryScore holds the score for one category (e.g., C1 Code Health).
type CategoryScore struct {
	Name      string     // Category identifier (e.g., "C1")
	Score     float64    // Weighted average of sub-scores (1-10)
	Weight    float64    // Weight in composite score
	SubScores []SubScore // Per-metric sub-scores
}

// SubScore holds the score for a single metric within a category.
type SubScore struct {
	MetricName string  // Metric identifier (e.g., "complexity_avg")
	RawValue   float64 // Original metric value
	Score      float64 // Interpolated score (1-10), negative if unavailable
	Weight     float64 // Weight within category
	Available  bool    // Whether the metric data was available
}
