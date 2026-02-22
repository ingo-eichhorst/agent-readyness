package discovery

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// generatedPattern matches the standard Go generated file comment.
// Must appear before the package declaration per Go convention.
var generatedPattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// classifyGoFile classifies a Go file by its filename.
// It checks for test files, underscore-prefixed files, and dot-prefixed files.
func classifyGoFile(name string) types.FileClass {
	if strings.HasSuffix(name, "_test.go") {
		return types.ClassTest
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return types.ClassExcluded
	}
	return types.ClassSource
}

// classifyPythonFile classifies a Python file by its filename.
// Test files match test_*.py or *_test.py patterns.
func classifyPythonFile(name string) types.FileClass {
	base := strings.TrimSuffix(name, ".py")
	if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test") {
		return types.ClassTest
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return types.ClassExcluded
	}
	return types.ClassSource
}

// classifyTypeScriptFile classifies a TypeScript file by its filename.
// Test files match *.test.ts, *.spec.ts, *.test.tsx, *.spec.tsx patterns.
func classifyTypeScriptFile(name string) types.FileClass {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".test.tsx") || strings.HasSuffix(lower, ".spec.tsx") {
		return types.ClassTest
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return types.ClassExcluded
	}
	return types.ClassSource
}

// isGeneratedFile checks whether a Go file contains a generated code comment
// before the package declaration. This handles files that have copyright headers
// before the generated comment (a common pattern with tools like stringer).
func isGeneratedFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Stop scanning at package declaration -- generated comment must be before it
		if strings.HasPrefix(line, "package ") {
			return false, nil
		}
		if generatedPattern.MatchString(line) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
