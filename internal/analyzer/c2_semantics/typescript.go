package c2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for TypeScript C2 metrics computation.
const (
	toPerKLOCTS          = 1000.0
	toPercentTS          = 100.0
	strictNullCheckPoints = 50.0
	chainDensityScale    = 10.0
	maxChainScore        = 50.0
)

// c2TypeScriptAnalyzer computes C2 (Semantic Explicitness) metrics for TypeScript code
// using Tree-sitter for parsing.
type c2TypeScriptAnalyzer struct{
	tsParser *parser.TreeSitterParser
}

// Newc2TypeScriptAnalyzer creates a TypeScript C2 analyzer with the given Tree-sitter parser.
func newC2TypeScriptAnalyzer(p *parser.TreeSitterParser) *c2TypeScriptAnalyzer {
	return &c2TypeScriptAnalyzer{tsParser: p}
}

// Analyze computes C2 metrics for a TypeScript AnalysisTarget.
func (a *c2TypeScriptAnalyzer) Analyze(target *types.AnalysisTarget) (*types.C2LanguageMetrics, error) {
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
		totalTypedElements int
		totalElements      int
		anyCount           int
		magicNumberCount   int
		totalLOC           int
		optionalChainCount int
		totalFunctions     int
	)

	for _, sf := range sourceFiles {
		content, err := os.ReadFile(sf.Path)
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := a.tsParser.ParseFile(types.LangTypeScript, ext, content)
		if err != nil {
			continue
		}

		root := tree.RootNode()
		lines := shared.CountLines(content)
		totalLOC += lines

		// C2-TS-01: Type annotation coverage
		typed, total, anyC, funcs := tsTypeAnnotations(root, content)
		totalTypedElements += typed
		totalElements += total
		anyCount += anyC
		totalFunctions += funcs

		// C2-TS-03: Magic numbers
		magicNumberCount += tsMagicNumbers(root, content)

		// C2-TS-04: Null safety -- count optional chaining
		optionalChainCount += tsOptionalChaining(root)

		tree.Close()
	}

	// Type annotation coverage: (typed - any_count) / total * 100
	if totalElements > 0 {
		effective := totalTypedElements - anyCount
		if effective < 0 {
			effective = 0
		}
		metrics.TypeAnnotationCoverage = float64(effective) / float64(totalElements) * toPercentTS
	}

	// Magic numbers
	metrics.MagicNumberCount = magicNumberCount
	if totalLOC > 0 {
		metrics.MagicNumberRatio = float64(magicNumberCount) / float64(totalLOC) * toPerKLOCTS
	}

	// C2-TS-02: tsconfig.json strict mode
	strictMode, strictNullChecks := tsDetectStrictMode(target.RootDir)
	if strictMode {
		metrics.TypeStrictness = 1
	}

	// C2-TS-04: Null safety score
	// Combination of strictNullChecks flag + optional chaining density
	nullSafetyScore := 0.0
	if strictNullChecks {
		nullSafetyScore += strictNullCheckPoints
	}
	// Optional chaining density: more usage = better
	if totalLOC > 0 {
		chainDensity := float64(optionalChainCount) / float64(totalLOC) * toPerKLOCTS
		// Scale: 0 chains/kLOC = 0 points, 5+ chains/kLOC = 50 points
		chainScore := chainDensity * chainDensityScale
		if chainScore > maxChainScore {
			chainScore = maxChainScore
		}
		nullSafetyScore += chainScore
	}
	metrics.NullSafety = nullSafetyScore

	metrics.TotalFunctions = totalFunctions
	metrics.LOC = totalLOC

	return metrics, nil
}

// tsTypeAnnotations counts type annotations in TypeScript functions.
// Returns: typedElements, totalElements, anyCount, functionCount.
func tsTypeAnnotations(root *tree_sitter.Node, content []byte) (int, int, int, int) {
	var typed, total, anyC, funcCount int

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		nodeKind := node.Kind()

		switch nodeKind {
		case "function_declaration", "method_definition":
			funcCount++

			// Check return type annotation
			total++ // One slot for return type
			returnType := node.ChildByFieldName("return_type")
			if returnType != nil {
				typed++
				if containsAnyType(returnType, content) {
					anyC++
				}
			}

			// Check parameters
			params := node.ChildByFieldName("parameters")
			if params != nil {
				countTSParams(params, content, &typed, &total, &anyC)
			}

		case "arrow_function":
			funcCount++

			// Check return type
			total++
			returnType := node.ChildByFieldName("return_type")
			if returnType != nil {
				typed++
				if containsAnyType(returnType, content) {
					anyC++
				}
			}

			// Check parameters
			params := node.ChildByFieldName("parameters")
			if params != nil {
				countTSParams(params, content, &typed, &total, &anyC)
			} else {
				// Single parameter arrow function (no parens)
				param := node.ChildByFieldName("parameter")
				if param != nil {
					total++
					// Check for type annotation on the single param
					typeAnno := param.ChildByFieldName("type")
					if typeAnno != nil {
						typed++
						if containsAnyType(typeAnno, content) {
							anyC++
						}
					}
				}
			}
		}
	})

	return typed, total, anyC, funcCount
}

// countTSParams counts typed/untyped parameters in a TypeScript parameter list.
func countTSParams(params *tree_sitter.Node, content []byte, typed, total, anyC *int) {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child == nil {
			continue
		}
		childKind := child.Kind()

		switch childKind {
		case "required_parameter", "optional_parameter", "rest_parameter":
			*total++
			typeAnno := child.ChildByFieldName("type")
			if typeAnno != nil {
				*typed++
				if containsAnyType(typeAnno, content) {
					*anyC++
				}
			}
		}
	}
}

// containsAnyType checks if a type annotation node contains explicit "any".
func containsAnyType(node *tree_sitter.Node, content []byte) bool {
	text := shared.NodeText(node, content)
	// Check for standalone "any" type
	return strings.Contains(text, "any")
}

// tsMagicNumbers counts magic numbers in TypeScript code.
func tsMagicNumbers(root *tree_sitter.Node, content []byte) int {
	count := 0

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		if node.Kind() != "number" {
			return
		}

		value := shared.NodeText(node, content)

		// Exclude common values
		if isTSCommonNumeric(value) {
			return
		}

		// Check parents for const declaration or enum
		parent := node.Parent()
		for parent != nil {
			parentKind := parent.Kind()
			if parentKind == "lexical_declaration" {
				// Check if it's a const
				for i := uint(0); i < parent.ChildCount(); i++ {
					child := parent.Child(i)
					if child != nil && shared.NodeText(child, content) == "const" {
						return
					}
				}
			}
			if parentKind == "enum_body" || parentKind == "enum_declaration" {
				return
			}
			parent = parent.Parent()
		}

		count++
	})

	return count
}

// tsOptionalChaining counts optional chaining (?.) usage.
func tsOptionalChaining(root *tree_sitter.Node) int {
	count := 0

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		nodeKind := node.Kind()
		// Tree-sitter represents optional chaining as member_expression with optional_chain
		if nodeKind == "optional_chain" {
			count++
		}
	})

	return count
}

// tsDetectStrictMode parses tsconfig.json and checks for strict mode settings.
// Returns (isStrict, hasStrictNullChecks).
func tsDetectStrictMode(rootDir string) (bool, bool) {
	tsconfigPath := filepath.Join(rootDir, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return false, false
	}

	var tsconfig struct {
		CompilerOptions struct {
			Strict              *bool `json:"strict"`
			StrictNullChecks    *bool `json:"strictNullChecks"`
			NoImplicitAny       *bool `json:"noImplicitAny"`
			StrictFunctionTypes *bool `json:"strictFunctionTypes"`
		} `json:"compilerOptions"`
	}

	if err := json.Unmarshal(data, &tsconfig); err != nil {
		return false, false
	}

	opts := tsconfig.CompilerOptions

	// strict == true means all strict flags are on
	if opts.Strict != nil && *opts.Strict {
		return true, true
	}

	// Check individual strict flags
	allStrict := boolTrue(opts.StrictNullChecks) &&
		boolTrue(opts.NoImplicitAny) &&
		boolTrue(opts.StrictFunctionTypes)

	hasNullChecks := boolTrue(opts.StrictNullChecks) || (opts.Strict != nil && *opts.Strict)

	return allStrict, hasNullChecks
}

// boolTrue returns true if the bool pointer is non-nil and true.
func boolTrue(b *bool) bool {
	return b != nil && *b
}

// isTSCommonNumeric returns true for commonly excluded numeric literals in TypeScript.
func isTSCommonNumeric(value string) bool {
	switch value {
	case "0", "1", "-1", "0.0", "1.0":
		return true
	}
	return false
}
