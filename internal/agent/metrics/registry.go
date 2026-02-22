package metrics

// registeredMetrics holds singleton instances of each metric.
var registeredMetrics = []Metric{
	newM1Consistency(),
	newM2Comprehension(),
	newM3Navigation(),
	newM4Identifiers(),
	newM5Documentation(),
}

// allMetrics returns all 5 MECE metrics for use within this package.
func allMetrics() []Metric {
	return registeredMetrics
}

// List returns all 5 MECE metrics.
// This is the exported entry point for packages that need to enumerate metrics.
func List() []Metric {
	return registeredMetrics
}

// getMetric returns a metric by ID, or nil if not found.
func getMetric(id string) Metric {
	for _, m := range registeredMetrics {
		if m.ID() == id {
			return m
		}
	}
	return nil
}
