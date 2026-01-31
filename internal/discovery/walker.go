package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/ingo/agent-readyness/pkg/types"
)

// skipDirs lists directory names that should be skipped during walking.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"testdata":     true,
}

// Walker discovers and classifies Go files in a directory tree.
type Walker struct{}

// NewWalker creates a new Walker instance.
func NewWalker() *Walker {
	return &Walker{}
}

// Discover walks rootDir recursively, discovers all .go files, classifies them,
// and returns a ScanResult with file lists and counts.
func (w *Walker) Discover(rootDir string) (*types.ScanResult, error) {
	// Validate rootDir exists and is a directory
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot access root directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", rootDir)
	}

	// Load .gitignore from root if present
	var gitIgnore *ignore.GitIgnore
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gitIgnore, err = ignore.CompileIgnoreFile(gitignorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse .gitignore: %w", err)
		}
	}

	result := &types.ScanResult{
		RootDir: rootDir,
	}

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories (starting with .)
			if strings.HasPrefix(name, ".") && name != "." {
				return fs.SkipDir
			}
			// Skip known excluded directories (except vendor -- we want to record vendor files)
			if skipDirs[name] {
				return fs.SkipDir
			}
			// Don't skip vendor dirs -- we walk into them to record files as excluded
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(name, ".go") {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		file := types.DiscoveredFile{
			Path:    path,
			RelPath: relPath,
		}

		// Check if in vendor directory
		if isVendorPath(relPath) {
			file.Class = types.ClassExcluded
			file.ExcludeReason = "vendor"
			result.Files = append(result.Files, file)
			result.VendorCount++
			result.TotalFiles++
			return nil
		}

		// Check gitignore
		if gitIgnore != nil && gitIgnore.MatchesPath(relPath) {
			file.Class = types.ClassExcluded
			file.ExcludeReason = "gitignore"
			result.Files = append(result.Files, file)
			result.GitignoreCount++
			result.TotalFiles++
			return nil
		}

		// Check if generated
		generated, err := IsGeneratedFile(path)
		if err != nil {
			return fmt.Errorf("failed to check generated status of %s: %w", relPath, err)
		}
		if generated {
			file.Class = types.ClassGenerated
			result.Files = append(result.Files, file)
			result.GeneratedCount++
			result.TotalFiles++
			return nil
		}

		// Classify by filename
		file.Class = ClassifyGoFile(name)
		result.Files = append(result.Files, file)
		result.TotalFiles++

		switch file.Class {
		case types.ClassSource:
			result.SourceCount++
		case types.ClassTest:
			result.TestCount++
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	return result, nil
}

// isVendorPath checks if a relative path is inside a vendor directory.
func isVendorPath(relPath string) bool {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for _, part := range parts {
		if part == "vendor" {
			return true
		}
	}
	return false
}
