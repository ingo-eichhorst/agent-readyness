package discovery

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// skipDirs lists directory names that should be skipped during walking.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"testdata":     true,
	"__pycache__":  true,
	"dist":         true,
	"build":        true,
	".venv":        true,
	"venv":         true,
	"env":          true,
}

// langExtensions maps file extensions to languages.
var langExtensions = map[string]types.Language{
	".go":  types.LangGo,
	".py":  types.LangPython,
	".ts":  types.LangTypeScript,
	".tsx": types.LangTypeScript,
}

// Walker discovers and classifies source files in a directory tree.
type Walker struct{}

// NewWalker creates a new Walker instance.
func NewWalker() *Walker {
	return &Walker{}
}

// Discover walks rootDir recursively, discovers all source files (.go, .py, .ts, .tsx),
// classifies them, and returns a ScanResult with file lists and counts.
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
		RootDir:     rootDir,
		PerLanguage: make(map[types.Language]int),
	}

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
			result.SkippedCount++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Symlink detection: skip symlinks before any other checks
		if d.Type()&fs.ModeSymlink != 0 {
			fmt.Fprintf(os.Stderr, "warning: skipping symlink %s\n", path)
			result.SymlinkCount++
			return nil
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

		// Determine language from extension
		ext := filepath.Ext(name)
		lang, supported := langExtensions[ext]
		if !supported {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to compute relative path: %v\n", path, err)
			result.SkippedCount++
			return nil
		}

		file := types.DiscoveredFile{
			Path:     path,
			RelPath:  relPath,
			Language: lang,
		}

		// Check if in vendor directory (Go-specific)
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

		// Check if generated (Go only)
		if lang == types.LangGo {
			generated, err := isGeneratedFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to check generated status: %v\n", relPath, err)
				result.SkippedCount++
				return nil
			}
			if generated {
				file.Class = types.ClassGenerated
				result.Files = append(result.Files, file)
				result.GeneratedCount++
				result.TotalFiles++
				return nil
			}
		}

		// Classify by filename based on language
		switch lang {
		case types.LangGo:
			file.Class = ClassifyGoFile(name)
		case types.LangPython:
			file.Class = classifyPythonFile(name)
		case types.LangTypeScript:
			file.Class = classifyTypeScriptFile(name)
		}

		result.Files = append(result.Files, file)
		result.TotalFiles++

		switch file.Class {
		case types.ClassSource:
			result.SourceCount++
			result.PerLanguage[lang]++
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

// DetectProjectLanguages checks the project root for language indicators and
// returns all languages detected.
func DetectProjectLanguages(rootDir string) []types.Language {
	var langs []types.Language

	// Go: go.mod or .go files
	if fileExists(filepath.Join(rootDir, "go.mod")) || hasFileWithExt(rootDir, ".go") {
		langs = append(langs, types.LangGo)
	}

	// Python: pyproject.toml, setup.py, setup.cfg, requirements.txt, or .py files
	pyIndicators := []string{"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt"}
	pyDetected := false
	for _, f := range pyIndicators {
		if fileExists(filepath.Join(rootDir, f)) {
			pyDetected = true
			break
		}
	}
	if !pyDetected {
		pyDetected = hasFileWithExt(rootDir, ".py")
	}
	if pyDetected {
		langs = append(langs, types.LangPython)
	}

	// TypeScript: tsconfig.json, .ts files, or package.json with typescript dep
	tsDetected := false
	if fileExists(filepath.Join(rootDir, "tsconfig.json")) {
		tsDetected = true
	}
	if !tsDetected {
		tsDetected = hasFileWithExt(rootDir, ".ts")
	}
	if !tsDetected {
		tsDetected = packageJSONHasTypeScript(filepath.Join(rootDir, "package.json"))
	}
	if tsDetected {
		langs = append(langs, types.LangTypeScript)
	}

	return langs
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

// fileExists returns true if path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// hasFileWithExt checks if any file with the given extension exists directly in dir.
func hasFileWithExt(dir string, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ext {
			return true
		}
	}
	return false
}

// packageJSONHasTypeScript checks if package.json has typescript in deps.
func packageJSONHasTypeScript(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}
	if _, ok := pkg.Dependencies["typescript"]; ok {
		return true
	}
	if _, ok := pkg.DevDependencies["typescript"]; ok {
		return true
	}
	return false
}
