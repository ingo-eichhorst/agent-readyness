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

	ctx := &walkContext{rootDir: rootDir, gitIgnore: gitIgnore, result: result}
	err = filepath.WalkDir(rootDir, ctx.visitEntry)
	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	return result, nil
}

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

type walkContext struct {
	rootDir   string
	gitIgnore *ignore.GitIgnore
	result    *types.ScanResult
}

func (ctx *walkContext) visitEntry(path string, d fs.DirEntry, err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", path, err)
		ctx.result.SkippedCount++
		if d != nil && d.IsDir() {
			return fs.SkipDir
		}
		return nil
	}

	if d.Type()&fs.ModeSymlink != 0 {
		fmt.Fprintf(os.Stderr, "warning: skipping symlink %s\n", path)
		ctx.result.SymlinkCount++
		return nil
	}

	if d.IsDir() {
		return ctx.visitDir(d.Name())
	}
	return ctx.visitFile(path, d.Name())
}

func (ctx *walkContext) visitDir(name string) error {
	if strings.HasPrefix(name, ".") && name != "." {
		return fs.SkipDir
	}
	if skipDirs[name] {
		return fs.SkipDir
	}
	return nil
}

func (ctx *walkContext) visitFile(path, name string) error {
	lang, supported := langExtensions[filepath.Ext(name)]
	if !supported {
		return nil
	}

	relPath, err := filepath.Rel(ctx.rootDir, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to compute relative path: %v\n", path, err)
		ctx.result.SkippedCount++
		return nil
	}

	file := types.DiscoveredFile{Path: path, RelPath: relPath, Language: lang}

	if isVendorPath(relPath) {
		ctx.addExcluded(file, "vendor", &ctx.result.VendorCount)
		return nil
	}
	if ctx.gitIgnore != nil && ctx.gitIgnore.MatchesPath(relPath) {
		ctx.addExcluded(file, "gitignore", &ctx.result.GitignoreCount)
		return nil
	}
	if lang == types.LangGo {
		if generated, err := isGeneratedFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: failed to check generated status: %v\n", relPath, err)
			ctx.result.SkippedCount++
			return nil
		} else if generated {
			file.Class = types.ClassGenerated
			ctx.result.Files = append(ctx.result.Files, file)
			ctx.result.GeneratedCount++
			ctx.result.TotalFiles++
			return nil
		}
	}

	file.Class = classifyByLanguage(lang, name)
	ctx.result.Files = append(ctx.result.Files, file)
	ctx.result.TotalFiles++
	if file.Class == types.ClassSource {
		ctx.result.SourceCount++
		ctx.result.PerLanguage[lang]++
	} else if file.Class == types.ClassTest {
		ctx.result.TestCount++
	}
	return nil
}

func (ctx *walkContext) addExcluded(file types.DiscoveredFile, reason string, counter *int) {
	file.Class = types.ClassExcluded
	file.ExcludeReason = reason
	ctx.result.Files = append(ctx.result.Files, file)
	*counter++
	ctx.result.TotalFiles++
}

func classifyByLanguage(lang types.Language, name string) types.FileClass {
	switch lang {
	case types.LangGo:
		return ClassifyGoFile(name)
	case types.LangPython:
		return classifyPythonFile(name)
	case types.LangTypeScript:
		return classifyTypeScriptFile(name)
	}
	return types.ClassSource
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
