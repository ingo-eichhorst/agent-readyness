package discovery

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/ingo/agent-readyness/pkg/types"
)

// generatedPattern matches the standard Go generated file comment.
// Must appear before the package declaration per Go convention.
var generatedPattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// ClassifyGoFile classifies a Go file by its filename.
// It checks for test files, underscore-prefixed files, and dot-prefixed files.
func ClassifyGoFile(name string) types.FileClass {
	if strings.HasSuffix(name, "_test.go") {
		return types.ClassTest
	}
	if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
		return types.ClassExcluded
	}
	return types.ClassSource
}

// IsGeneratedFile checks whether a Go file contains a generated code comment
// before the package declaration. This handles files that have copyright headers
// before the generated comment (a common pattern with tools like stringer).
func IsGeneratedFile(path string) (bool, error) {
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
