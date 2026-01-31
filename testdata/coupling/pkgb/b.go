package pkgb

// Hello returns a greeting. pkgb has no intra-module imports (efferent=0).
// pkgb is imported by pkga, so afferent=1.
func Hello() string {
	return "hello"
}
