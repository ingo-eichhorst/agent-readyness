package c2

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for Python C2 metrics computation.
const (
	toPerKLOCPy = 1000.0
	toPercentPy = 100.0
)

// c2PythonAnalyzer computes C2 (Semantic Explicitness) metrics for Python code
// using Tree-sitter for parsing.
type c2PythonAnalyzer struct {
	tsParser *parser.TreeSitterParser
}

// Newc2PythonAnalyzer creates a Python C2 analyzer with the given Tree-sitter parser.
func newC2PythonAnalyzer(p *parser.TreeSitterParser) *c2PythonAnalyzer {
	return &c2PythonAnalyzer{tsParser: p}
}

// Analyze computes C2 metrics for a Python AnalysisTarget.
func (a *c2PythonAnalyzer) Analyze(target *types.AnalysisTarget) (*types.C2LanguageMetrics, error) {
	metrics := &types.C2LanguageMetrics{}

	sourceFiles := pyFilterSourceFiles(target.Files)
	if len(sourceFiles) == 0 {
		return metrics, nil
	}

	stats := pyAnalyzeSourceFiles(a.tsParser, sourceFiles)
	pyPopulateMetrics(metrics, stats, target.RootDir)

	return metrics, nil
}

type pyAnalysisStats struct {
	totalAnnotatedParams  int
	totalAnnotatedReturns int
	totalParams           int
	totalFunctions        int
	totalIdentifiers      int
	consistentNames       int
	magicNumberCount      int
	totalLOC              int
}

func pyFilterSourceFiles(files []types.SourceFile) []types.SourceFile {
	var sourceFiles []types.SourceFile
	for _, sf := range files {
		if sf.Class == types.ClassSource {
			sourceFiles = append(sourceFiles, sf)
		}
	}
	return sourceFiles
}

func pyAnalyzeSourceFiles(tsParser *parser.TreeSitterParser, sourceFiles []types.SourceFile) pyAnalysisStats {
	var stats pyAnalysisStats
	for _, sf := range sourceFiles {
		content, err := os.ReadFile(sf.Path)
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := tsParser.ParseFile(types.LangPython, ext, content)
		if err != nil {
			continue
		}
		root := tree.RootNode()
		stats.totalLOC += shared.CountLines(content)
		ap, ar, tp, tf := pyTypeAnnotations(root, content)
		stats.totalAnnotatedParams += ap
		stats.totalAnnotatedReturns += ar
		stats.totalParams += tp
		stats.totalFunctions += tf
		consistent, total := pyNamingConsistency(root, content)
		stats.consistentNames += consistent
		stats.totalIdentifiers += total
		stats.magicNumberCount += pyMagicNumbers(root, content)
		tree.Close()
	}
	return stats
}

func pyPopulateMetrics(metrics *types.C2LanguageMetrics, stats pyAnalysisStats, rootDir string) {
	denominator := stats.totalParams + stats.totalFunctions
	if denominator > 0 {
		metrics.TypeAnnotationCoverage = float64(stats.totalAnnotatedParams+stats.totalAnnotatedReturns) / float64(denominator) * toPercentPy
	}
	if stats.totalIdentifiers > 0 {
		metrics.NamingConsistency = float64(stats.consistentNames) / float64(stats.totalIdentifiers) * toPercentPy
	}
	metrics.MagicNumberCount = stats.magicNumberCount
	if stats.totalLOC > 0 {
		metrics.MagicNumberRatio = float64(stats.magicNumberCount) / float64(stats.totalLOC) * toPerKLOCPy
	}
	metrics.TypeStrictness = pyDetectTypeChecker(rootDir)
	metrics.TotalFunctions = stats.totalFunctions
	metrics.TotalIdentifiers = stats.totalIdentifiers
	metrics.LOC = stats.totalLOC
}

// pyTypeAnnotations counts type annotations in Python functions.
// Returns: annotatedParams, annotatedReturns, totalParams, totalFunctions.
func pyTypeAnnotations(root *tree_sitter.Node, content []byte) (int, int, int, int) {
	var annotatedParams, annotatedReturns, totalParams, totalFunctions int

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		if node.Kind() != "function_definition" {
			return
		}
		totalFunctions++
		if returnType := node.ChildByFieldName("return_type"); returnType != nil {
			annotatedReturns++
		}
		params := node.ChildByFieldName("parameters")
		if params == nil {
			return
		}
		pyProcessParameters(params, content, &annotatedParams, &totalParams)
	})

	return annotatedParams, annotatedReturns, totalParams, totalFunctions
}

func pyProcessParameters(params *tree_sitter.Node, content []byte, annotatedParams, totalParams *int) {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child == nil {
			continue
		}
		pyProcessParameter(child, content, annotatedParams, totalParams)
	}
}

func pyProcessParameter(child *tree_sitter.Node, content []byte, annotatedParams, totalParams *int) {
	childKind := child.Kind()
	switch childKind {
	case "identifier":
		paramName := shared.NodeText(child, content)
		if paramName == "self" || paramName == "cls" {
			return
		}
		*totalParams++
	case "typed_parameter":
		if pyIsSelfOrCls(child, content) {
			return
		}
		*totalParams++
		*annotatedParams++
	case "default_parameter":
		if pyIsSelfOrCls(child, content) {
			return
		}
		*totalParams++
	case "typed_default_parameter":
		if pyIsSelfOrCls(child, content) {
			return
		}
		*totalParams++
		*annotatedParams++
	case "list_splat_pattern", "dictionary_splat_pattern":
		*totalParams++
	}
}

func pyIsSelfOrCls(child *tree_sitter.Node, content []byte) bool {
	nameNode := child.ChildByFieldName("name")
	if nameNode != nil {
		paramName := shared.NodeText(nameNode, content)
		return paramName == "self" || paramName == "cls"
	}
	return false
}

// pyNamingConsistency checks Python naming conventions (PEP 8).
func pyNamingConsistency(root *tree_sitter.Node, content []byte) (int, int) {
	var consistent, total int

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		parent := node.Parent()
		if parent == nil {
			return
		}
		parentKind := parent.Kind()

		switch {
		case nodeType == "identifier" && parentKind == "function_definition":
			// Function/method names: must be snake_case
			// Check if this identifier is the "name" field of the function_definition
			nameNode := parent.ChildByFieldName("name")
			if nameNode == nil || nameNode.Id() != node.Id() {
				return
			}
			name := shared.NodeText(node, content)
			if name == "" || name == "_" || len(name) <= 1 {
				return
			}
			// Skip dunder methods
			if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") {
				return
			}
			total++
			if isSnakeCase(name) {
				consistent++
			}

		case nodeType == "identifier" && parentKind == "class_definition":
			// Class names: must be CamelCase
			nameNode := parent.ChildByFieldName("name")
			if nameNode == nil || nameNode.Id() != node.Id() {
				return
			}
			name := shared.NodeText(node, content)
			if name == "" || len(name) <= 1 {
				return
			}
			total++
			if isCamelCase(name) {
				consistent++
			}
		}
	})

	return consistent, total
}

// pyMagicNumbers counts magic numbers in Python code.
func pyMagicNumbers(root *tree_sitter.Node, content []byte) int {
	count := 0

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		if nodeType != "integer" && nodeType != "float" {
			return
		}

		value := shared.NodeText(node, content)

		// Exclude common values
		if isPyCommonNumeric(value) {
			return
		}

		// Exclude: inside assignment to UPPER_CASE names
		parent := node.Parent()
		if parent != nil && isUpperCaseAssignment(parent, content) {
			return
		}

		// Exclude: inside subscript (list/dict indices)
		if parent != nil && parent.Kind() == "subscript" {
			return
		}

		count++
	})

	return count
}

// pyDetectTypeChecker checks for mypy/pyright configuration files.
func pyDetectTypeChecker(rootDir string) float64 {
	// Direct config files
	directConfigs := []string{
		"mypy.ini",
		".mypy.ini",
		"pyrightconfig.json",
	}
	for _, name := range directConfigs {
		if _, err := os.Stat(filepath.Join(rootDir, name)); err == nil {
			return 1
		}
	}

	// setup.cfg with [mypy] section
	if hasINISection(filepath.Join(rootDir, "setup.cfg"), "[mypy]") {
		return 1
	}

	// pyproject.toml with [tool.mypy] or [tool.pyright]
	pyprojectPath := filepath.Join(rootDir, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		content := string(data)
		if strings.Contains(content, "[tool.mypy]") || strings.Contains(content, "[tool.pyright]") {
			return 1
		}
	}

	return 0
}

// hasINISection checks if a file contains a given INI-style section header.
func hasINISection(path string, section string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), section)
}

// isSnakeCase checks if a name follows snake_case convention.
var snakeCasePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

func isSnakeCase(name string) bool {
	return snakeCasePattern.MatchString(name)
}

// isCamelCase checks if a name starts with uppercase (CamelCase/PascalCase).
func isCamelCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

// isPyCommonNumeric returns true for commonly excluded numeric literals.
func isPyCommonNumeric(value string) bool {
	switch value {
	case "0", "1", "-1", "2", "100", "0.0", "1.0":
		return true
	}
	return false
}

// isUpperCaseAssignment checks if a node is inside an assignment to an UPPER_CASE name.
var upperCasePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func isUpperCaseAssignment(node *tree_sitter.Node, content []byte) bool {
	// Walk up to find assignment
	current := node
	for current != nil {
		if current.Kind() == "assignment" {
			left := current.ChildByFieldName("left")
			if left != nil && left.Kind() == "identifier" {
				name := shared.NodeText(left, content)
				return upperCasePattern.MatchString(name)
			}
		}
		current = current.Parent()
	}
	return false
}
