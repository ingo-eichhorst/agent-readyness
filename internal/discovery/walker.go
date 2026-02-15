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
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("cannot access root directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", rootDir)
	}

	gitIgnore, err := loadGitIgnore(rootDir)
	if err != nil {
		return nil, err
	}

	result := &types.ScanResult{
		RootDir:     rootDir,
		PerLanguage: make(map[types.Language]int),
	}

	wc := &walkContext{rootDir: rootDir, gitIgnore: gitIgnore, result: result}
	if err := filepath.WalkDir(rootDir, wc.visit); err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}
	return result, nil
}

// loadGitIgnore loads .gitignore from root if present.
func loadGitIgnore(rootDir string) (*ignore.GitIgnore, error) {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		return nil, nil
	}
	gi, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse .gitignore: %w", err)
	}
	return gi, nil
}

// walkContext holds state for the directory walk.
type walkContext struct {
	rootDir   string
	gitIgnore *ignore.GitIgnore
	result    *types.ScanResult
}

// visit is the WalkDir callback that classifies each entry.
func (wc *walkContext) visit(path string, d fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
		wc.result.SkippedCount++
		if d != nil && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}

	if d.Type()&fs.ModeSymlink != 0 {
		fmt.Fprintf(os.Stderr, "warning: skipping symlink %s\n", path)
		wc.result.SymlinkCount++
		return nil
	}

	if d.IsDir() {
		return wc.visitDir(d.Name())
	}
	return wc.visitFile(path, d.Name())
}

// visitDir decides whether to enter or skip a directory.
func (wc *walkContext) visitDir(name string) error {
	if strings.HasPrefix(name, ".") && name != "." {
		return fs.SkipDir
	}
	if skipDirs[name] {
		return fs.SkipDir
	}
	return nil
}

// visitFile classifies a source file and adds it to results.
func (wc *walkContext) visitFile(path, name string) error {
	lang, supported := langExtensions[filepath.Ext(name)]
	if !supported {
		return nil
	}

	relPath, err := filepath.Rel(wc.rootDir, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to compute relative path: %v\n", path, err)
		wc.result.SkippedCount++
		return nil
	}

	file := types.DiscoveredFile{Path: path, RelPath: relPath, Language: lang}

	if excluded := wc.checkExclusions(&file, path, relPath, lang); excluded {
		return nil
	}

	file.Class = classifyByLanguage(lang, name)
	wc.result.Files = append(wc.result.Files, file)
	wc.result.TotalFiles++
	if file.Class == types.ClassSource {
		wc.result.SourceCount++
		wc.result.PerLanguage[lang]++
	} else if file.Class == types.ClassTest {
		wc.result.TestCount++
	}
	return nil
}

// checkExclusions checks vendor, gitignore, and generated-file exclusions.
// Returns true if the file was excluded and already added to results.
func (wc *walkContext) checkExclusions(file *types.DiscoveredFile, path, relPath string, lang types.Language) bool {
	if isVendorPath(relPath) {
		file.Class = types.ClassExcluded
		file.ExcludeReason = "vendor"
		wc.result.Files = append(wc.result.Files, *file)
		wc.result.VendorCount++
		wc.result.TotalFiles++
		return true
	}
	if wc.gitIgnore != nil && wc.gitIgnore.MatchesPath(relPath) {
		file.Class = types.ClassExcluded
		file.ExcludeReason = "gitignore"
		wc.result.Files = append(wc.result.Files, *file)
		wc.result.GitignoreCount++
		wc.result.TotalFiles++
		return true
	}
	if lang == types.LangGo {
		generated, err := isGeneratedFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to check generated status: %v\n", relPath, err)
			wc.result.SkippedCount++
			return true
		}
		if generated {
			file.Class = types.ClassGenerated
			wc.result.Files = append(wc.result.Files, *file)
			wc.result.GeneratedCount++
			wc.result.TotalFiles++
			return true
		}
	}
	return false
}

// classifyByLanguage classifies a file based on its language and filename.
func classifyByLanguage(lang types.Language, name string) types.FileClass {
	switch lang {
	case types.LangGo:
		return ClassifyGoFile(name)
	case types.LangPython:
		return classifyPythonFile(name)
	case types.LangTypeScript:
		return classifyTypeScriptFile(name)
	default:
		return types.ClassSource
	}
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
