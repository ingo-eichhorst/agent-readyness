package metrics

// allMetrics holds singleton instances of each metric.
var allMetrics = []Metric{
	NewM1Consistency(),
	NewM2Comprehension(),
	NewM3Navigation(),
	NewM4Identifiers(),
	NewM5Documentation(),
}

// AllMetrics returns all 5 MECE metrics.
func AllMetrics() []Metric {
	return allMetrics
}

// GetMetric returns a metric by ID, or nil if not found.
func GetMetric(id string) Metric {
	for _, m := range allMetrics {
		if m.ID() == id {
			return m
		}
	}
	return nil
}
