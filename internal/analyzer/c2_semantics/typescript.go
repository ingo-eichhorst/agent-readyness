package c2

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/ingo-eichhorst/agent-readyness/internal/analyzer/shared"
	"github.com/ingo-eichhorst/agent-readyness/internal/parser"
	"github.com/ingo-eichhorst/agent-readyness/pkg/types"
)

// Constants for TypeScript C2 metrics computation.
const (
	toPerKLOCTS           = 1000.0
	toPercentTS           = 100.0
	strictNullCheckPoints = 50.0
	chainDensityScale     = 10.0
	maxChainScore         = 50.0
	// anyRateAnnotationScale converts any-usages-per-KLOC to a coverage penalty.
	// e.g. 5 any/KLOC → 100 - 15 = 85%; 33 any/KLOC → 0%.
	anyRateAnnotationScale = 3.0
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
type tsAccumulator struct {
	anyCount           int // total explicit `any` type usages (all contexts)
	magicNumberCount   int
	totalLOC           int
	optionalChainCount int
	totalFunctions     int
	consistentNames    int // identifiers following TS naming conventions
	totalNamingChecked int // total identifiers checked for naming conventions
}

func (a *c2TypeScriptAnalyzer) Analyze(target *types.AnalysisTarget) (*types.C2LanguageMetrics, error) {
	var sourceFiles []types.SourceFile
	for _, sf := range target.Files {
		if sf.Class == types.ClassSource {
			sourceFiles = append(sourceFiles, sf)
		}
	}
	if len(sourceFiles) == 0 {
		return &types.C2LanguageMetrics{}, nil
	}

	acc := a.accumulateTSMetrics(sourceFiles)
	return acc.buildMetrics(target.RootDir), nil
}

func (a *c2TypeScriptAnalyzer) accumulateTSMetrics(sourceFiles []types.SourceFile) tsAccumulator {
	var acc tsAccumulator
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
		acc.totalLOC += shared.CountLines(content)
		acc.anyCount += tsCountAnyUsages(root, content)
		acc.totalFunctions += tsCountFunctions(root)
		acc.magicNumberCount += tsMagicNumbers(root, content)
		acc.optionalChainCount += tsOptionalChaining(root)
		consistent, total := tsNamingConsistency(root, content)
		acc.consistentNames += consistent
		acc.totalNamingChecked += total
		tree.Close()
	}
	return acc
}

func (acc *tsAccumulator) buildMetrics(rootDir string) *types.C2LanguageMetrics {
	metrics := &types.C2LanguageMetrics{
		// TypeScript naming consistency is not yet implemented.
		// Mark as unavailable so it's excluded from the weighted average
		// rather than penalizing TS repos with a default of 0.
		Unavailable: map[string]bool{
			"naming_consistency": true,
		},
	}

	// TypeAnnotationCoverage: inverse of any-usage rate per KLOC.
	// TypeScript uses inference, so counting missing annotations is misleading.
	// High-quality TS code avoids explicit `any`; this metric is fair regardless
	// of whether inference is used.
	if acc.totalLOC > 0 {
		anyRatePerKLOC := float64(acc.anyCount) / float64(acc.totalLOC) * toPerKLOCTS
		coverage := toPercentTS - anyRatePerKLOC*anyRateAnnotationScale
		if coverage < 0 {
			coverage = 0
		}
		metrics.TypeAnnotationCoverage = coverage
	}

	metrics.MagicNumberCount = acc.magicNumberCount
	if acc.totalLOC > 0 {
		metrics.MagicNumberRatio = float64(acc.magicNumberCount) / float64(acc.totalLOC) * toPerKLOCTS
	}

	strictMode, strictNullChecks := tsDetectStrictMode(rootDir)
	if strictMode {
		metrics.TypeStrictness = 1
	}

	metrics.NullSafety = tsNullSafetyScore(strictNullChecks, acc.optionalChainCount, acc.totalLOC)
	if acc.totalNamingChecked > 0 {
		metrics.NamingConsistency = float64(acc.consistentNames) / float64(acc.totalNamingChecked) * toPercentTS
	}
	metrics.TotalFunctions = acc.totalFunctions
	metrics.TotalIdentifiers = acc.totalNamingChecked
	metrics.LOC = acc.totalLOC
	return metrics
}

func tsNullSafetyScore(strictNullChecks bool, optionalChainCount, totalLOC int) float64 {
	score := 0.0
	if strictNullChecks {
		score += strictNullCheckPoints
	}
	if totalLOC > 0 {
		chainDensity := float64(optionalChainCount) / float64(totalLOC) * toPerKLOCTS
		chainScore := chainDensity * chainDensityScale
		if chainScore > maxChainScore {
			chainScore = maxChainScore
		}
		score += chainScore
	}
	return score
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

// tsCountAnyUsages counts all explicit `any` type usages in a TypeScript file.
// It walks the entire AST looking for predefined_type nodes with text "any",
// covering variable declarations, function annotations, and other contexts.
func tsCountAnyUsages(root *tree_sitter.Node, content []byte) int {
	count := 0
	shared.WalkTree(root, func(node *tree_sitter.Node) {
		if node.Kind() == "predefined_type" && shared.NodeText(node, content) == "any" {
			count++
		}
	})
	return count
}

// tsCountFunctions counts function declarations, method definitions, and arrow functions.
func tsCountFunctions(root *tree_sitter.Node) int {
	count := 0
	shared.WalkTree(root, func(node *tree_sitter.Node) {
		switch node.Kind() {
		case "function_declaration", "method_definition", "arrow_function":
			count++
		}
	})
	return count
}

// tsDetectStrictMode scans all tsconfig*.json files under rootDir for strict mode settings.
// This handles monorepos where strict: true may live in a nested or referenced config.
// Returns (isStrict, hasStrictNullChecks).
func tsDetectStrictMode(rootDir string) (bool, bool) {
	var foundStrict, foundNullChecks bool

	_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			// Skip node_modules and hidden directories to avoid false positives
			// and keep the walk fast.
			if name == "node_modules" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasPrefix(base, "tsconfig") || !strings.HasSuffix(base, ".json") {
			return nil
		}
		isStrict, hasNullChecks := tsParseOneConfig(path)
		if isStrict {
			foundStrict = true
			foundNullChecks = foundNullChecks || hasNullChecks
		}
		return nil
	})

	return foundStrict, foundNullChecks
}

// tsParseOneConfig reads a single tsconfig JSON file and returns strict mode flags.
func tsParseOneConfig(path string) (isStrict bool, hasStrictNullChecks bool) {
	data, err := os.ReadFile(path)
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

	// strict == true enables all strict flags.
	if opts.Strict != nil && *opts.Strict {
		return true, true
	}

	// Check individual strict flags.
	allStrict := boolTrue(opts.StrictNullChecks) &&
		boolTrue(opts.NoImplicitAny) &&
		boolTrue(opts.StrictFunctionTypes)

	return allStrict, boolTrue(opts.StrictNullChecks)
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

// tsNamingConsistency checks TypeScript naming conventions:
//   - Functions and methods: camelCase (lowercase first letter)
//   - Classes, interfaces, and type aliases: PascalCase (uppercase first letter)
//
// Returns (consistent, total) counts.
func tsNamingConsistency(root *tree_sitter.Node, content []byte) (int, int) {
	var consistent, total int

	shared.WalkTree(root, func(node *tree_sitter.Node) {
		switch node.Kind() {
		case "function_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := shared.NodeText(nameNode, content)
			if tsShouldSkipName(name) {
				return
			}
			total++
			if isTSCamelCase(name) {
				consistent++
			}

		case "method_definition":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := shared.NodeText(nameNode, content)
			// Skip constructor and well-known lifecycle names.
			if name == "constructor" || tsShouldSkipName(name) {
				return
			}
			total++
			if isTSCamelCase(name) {
				consistent++
			}

		case "class_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := shared.NodeText(nameNode, content)
			if tsShouldSkipName(name) {
				return
			}
			total++
			if isTSPascalCase(name) {
				consistent++
			}

		case "interface_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := shared.NodeText(nameNode, content)
			if tsShouldSkipName(name) {
				return
			}
			total++
			if isTSPascalCase(name) {
				consistent++
			}

		case "type_alias_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return
			}
			name := shared.NodeText(nameNode, content)
			if tsShouldSkipName(name) {
				return
			}
			total++
			if isTSPascalCase(name) {
				consistent++
			}
		}
	})

	return consistent, total
}

// isTSCamelCase returns true if the name follows camelCase convention:
// optional leading underscore(s) for private members, then a lowercase letter,
// then no further underscores.
func isTSCamelCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Strip optional leading underscores (private-by-convention prefix).
	stripped := strings.TrimLeft(name, "_")
	if stripped == "" {
		return false // e.g. "___"
	}
	return unicode.IsLower(rune(stripped[0])) && !strings.Contains(stripped, "_")
}

// isTSPascalCase returns true if the name follows PascalCase convention:
// starts with an uppercase letter and contains no underscores.
func isTSPascalCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0])) && !strings.Contains(name, "_")
}

// tsShouldSkipName returns true if the identifier should be excluded from naming checks.
func tsShouldSkipName(name string) bool {
	return len(name) <= 1 || name == "_"
}
