package analyzer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo/agent-readyness/internal/parser"
	"github.com/ingo/agent-readyness/pkg/types"
)

// C2PythonAnalyzer computes C2 (Semantic Explicitness) metrics for Python code
// using Tree-sitter for parsing.
type C2PythonAnalyzer struct {
	tsParser *parser.TreeSitterParser
}

// NewC2PythonAnalyzer creates a Python C2 analyzer with the given Tree-sitter parser.
func NewC2PythonAnalyzer(p *parser.TreeSitterParser) *C2PythonAnalyzer {
	return &C2PythonAnalyzer{tsParser: p}
}

// Analyze computes C2 metrics for a Python AnalysisTarget.
func (a *C2PythonAnalyzer) Analyze(target *types.AnalysisTarget) (*types.C2LanguageMetrics, error) {
	metrics := &types.C2LanguageMetrics{}

	// Filter to source files only (skip test files)
	var sourceFiles []types.SourceFile
	for _, sf := range target.Files {
		if sf.Class == types.ClassSource {
			sourceFiles = append(sourceFiles, sf)
		}
	}

	if len(sourceFiles) == 0 {
		return metrics, nil
	}

	var (
		totalAnnotatedParams int
		totalAnnotatedReturns int
		totalParams          int
		totalFunctions       int
		totalIdentifiers     int
		consistentNames      int
		magicNumberCount     int
		totalLOC             int
	)

	for _, sf := range sourceFiles {
		content, err := os.ReadFile(sf.Path)
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := a.tsParser.ParseFile(types.LangPython, ext, content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		lines := CountLines(content)
		totalLOC += lines

		// C2-PY-01: Type annotation coverage
		ap, ar, tp, tf := pyTypeAnnotations(root, content)
		totalAnnotatedParams += ap
		totalAnnotatedReturns += ar
		totalParams += tp
		totalFunctions += tf

		// C2-PY-02: Naming consistency (PEP 8)
		consistent, total := pyNamingConsistency(root, content)
		consistentNames += consistent
		totalIdentifiers += total

		// C2-PY-03: Magic numbers
		magicNumberCount += pyMagicNumbers(root, content)

		tree.Close()
	}

	// Type annotation coverage score
	denominator := totalParams + totalFunctions
	if denominator > 0 {
		metrics.TypeAnnotationCoverage = float64(totalAnnotatedParams+totalAnnotatedReturns) / float64(denominator) * 100
	}

	// Naming consistency score
	if totalIdentifiers > 0 {
		metrics.NamingConsistency = float64(consistentNames) / float64(totalIdentifiers) * 100
	}

	// Magic number ratio per 1000 LOC
	metrics.MagicNumberCount = magicNumberCount
	if totalLOC > 0 {
		metrics.MagicNumberRatio = float64(magicNumberCount) / float64(totalLOC) * 1000
	}

	// C2-PY-04: mypy/pyright config detection
	metrics.TypeStrictness = pyDetectTypeChecker(target.RootDir)

	metrics.TotalFunctions = totalFunctions
	metrics.TotalIdentifiers = totalIdentifiers
	metrics.LOC = totalLOC

	return metrics, nil
}

// pyTypeAnnotations counts type annotations in Python functions.
// Returns: annotatedParams, annotatedReturns, totalParams, totalFunctions.
func pyTypeAnnotations(root *tree_sitter.Node, content []byte) (int, int, int, int) {
	var annotatedParams, annotatedReturns, totalParams, totalFunctions int

	WalkTree(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		if nodeType != "function_definition" {
			return
		}

		totalFunctions++

		// Check return type annotation
		if returnType := node.ChildByFieldName("return_type"); returnType != nil {
			annotatedReturns++
		}

		// Check parameters
		params := node.ChildByFieldName("parameters")
		if params == nil {
			return
		}

		for i := uint(0); i < params.ChildCount(); i++ {
			child := params.Child(i)
			if child == nil {
				continue
			}

			childKind := child.Kind()
			switch childKind {
			case "identifier":
				// Plain parameter without type annotation
				paramName := NodeText(child, content)
				if paramName == "self" || paramName == "cls" {
					continue
				}
				totalParams++

			case "typed_parameter":
				// Parameter with type annotation
				// Check if it's self/cls
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					paramName := NodeText(nameNode, content)
					if paramName == "self" || paramName == "cls" {
						continue
					}
				}
				totalParams++
				annotatedParams++

			case "default_parameter":
				// Default parameter -- check if it has a type annotation
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					paramName := NodeText(nameNode, content)
					if paramName == "self" || paramName == "cls" {
						continue
					}
				}
				totalParams++
				// default_parameter with type becomes typed_default_parameter in tree-sitter

			case "typed_default_parameter":
				nameNode := child.ChildByFieldName("name")
				if nameNode != nil {
					paramName := NodeText(nameNode, content)
					if paramName == "self" || paramName == "cls" {
						continue
					}
				}
				totalParams++
				annotatedParams++

			case "list_splat_pattern", "dictionary_splat_pattern":
				// *args, **kwargs
				totalParams++
			}
		}
	})

	return annotatedParams, annotatedReturns, totalParams, totalFunctions
}

// pyNamingConsistency checks Python naming conventions (PEP 8).
func pyNamingConsistency(root *tree_sitter.Node, content []byte) (int, int) {
	var consistent, total int

	WalkTree(root, func(node *tree_sitter.Node) {
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
			name := NodeText(node, content)
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
			name := NodeText(node, content)
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

	WalkTree(root, func(node *tree_sitter.Node) {
		nodeType := node.Kind()
		if nodeType != "integer" && nodeType != "float" {
			return
		}

		value := NodeText(node, content)

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
				name := NodeText(left, content)
				return upperCasePattern.MatchString(name)
			}
		}
		current = current.Parent()
	}
	return false
}
