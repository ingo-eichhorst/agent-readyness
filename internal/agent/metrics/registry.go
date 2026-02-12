package metrics

// allMetrics holds singleton instances of each metric.
var allMetrics = []Metric{
	newM1Consistency(),
	newM2Comprehension(),
	newM3Navigation(),
	newM4Identifiers(),
	newM5Documentation(),
}

// AllMetrics returns all 5 MECE metrics.
func AllMetrics() []Metric {
	return allMetrics
}

// getMetric returns a metric by ID, or nil if not found.
func getMetric(id string) Metric {
	for _, m := range allMetrics {
		if m.ID() == id {
			return m
		}
	}
	return nil
}
