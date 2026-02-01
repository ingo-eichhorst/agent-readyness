// Package parser provides source code parsing for Go, Python, and TypeScript.
//
// Tree-sitter parsers require CGO_ENABLED=1. The TreeSitterParser provides
// pooled parsers for Python, TypeScript, and TSX. Every Tree and Parser must
// be explicitly closed to avoid memory leaks.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/ingo/agent-readyness/pkg/types"
)

// ParsedTreeSitterFile holds a parsed Tree-sitter syntax tree with its source content.
// Caller must call Tree.Close() when done, or use CloseAll.
type ParsedTreeSitterFile struct {
	Path     string
	RelPath  string
	Tree     *tree_sitter.Tree
	Content  []byte
	Language types.Language
}

// TreeSitterParser holds pooled Tree-sitter parsers for Python, TypeScript, and TSX.
// Tree-sitter parsers are NOT thread-safe, so all parse operations are serialized
// via a mutex. Trees returned from parsing are safe to use concurrently after parsing.
type TreeSitterParser struct {
	mu           sync.Mutex
	pythonParser *tree_sitter.Parser
	tsParser     *tree_sitter.Parser
	tsxParser    *tree_sitter.Parser
}

// NewTreeSitterParser creates parsers for Python, TypeScript, and TSX.
// Returns an error if any language fails to initialize.
func NewTreeSitterParser() (*TreeSitterParser, error) {
	pyParser := tree_sitter.NewParser()
	pyLang := tree_sitter.NewLanguage(tree_sitter_python.Language())
	if err := pyParser.SetLanguage(pyLang); err != nil {
		pyParser.Close()
		return nil, fmt.Errorf("set python language: %w", err)
	}

	tsParser := tree_sitter.NewParser()
	tsLang := tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTypescript())
	if err := tsParser.SetLanguage(tsLang); err != nil {
		pyParser.Close()
		tsParser.Close()
		return nil, fmt.Errorf("set typescript language: %w", err)
	}

	tsxParser := tree_sitter.NewParser()
	tsxLang := tree_sitter.NewLanguage(tree_sitter_typescript.LanguageTSX())
	if err := tsxParser.SetLanguage(tsxLang); err != nil {
		pyParser.Close()
		tsParser.Close()
		tsxParser.Close()
		return nil, fmt.Errorf("set tsx language: %w", err)
	}

	return &TreeSitterParser{
		pythonParser: pyParser,
		tsParser:     tsParser,
		tsxParser:    tsxParser,
	}, nil
}

// Close releases all parser resources. Must be called when done.
func (p *TreeSitterParser) Close() {
	if p.pythonParser != nil {
		p.pythonParser.Close()
	}
	if p.tsParser != nil {
		p.tsParser.Close()
	}
	if p.tsxParser != nil {
		p.tsxParser.Close()
	}
}

// ParseFile parses source content for the given language and file extension.
// The ext parameter is used to distinguish .ts from .tsx for TypeScript.
// Returns a Tree that the caller must close.
// This method is thread-safe; parsing is serialized internally.
func (p *TreeSitterParser) ParseFile(lang types.Language, ext string, content []byte) (*tree_sitter.Tree, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var parser *tree_sitter.Parser

	switch lang {
	case types.LangPython:
		parser = p.pythonParser
	case types.LangTypeScript:
		if ext == ".tsx" {
			parser = p.tsxParser
		} else {
			parser = p.tsParser
		}
	default:
		return nil, fmt.Errorf("unsupported language for Tree-sitter: %s", lang)
	}

	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter parse returned nil")
	}

	return tree, nil
}

// ParseTargetFiles parses all source files in an AnalysisTarget, returning
// parsed trees. Caller must close all returned trees (use CloseAll helper).
// Only parses Python and TypeScript files; Go files are skipped.
func (p *TreeSitterParser) ParseTargetFiles(target *types.AnalysisTarget) ([]*ParsedTreeSitterFile, error) {
	if target.Language == types.LangGo {
		return nil, fmt.Errorf("Go files should be parsed with go/packages, not Tree-sitter")
	}

	var results []*ParsedTreeSitterFile

	for _, sf := range target.Files {
		if sf.Class != types.ClassSource && sf.Class != types.ClassTest {
			continue
		}

		content, err := os.ReadFile(sf.Path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", sf.RelPath, err)
		}

		ext := strings.ToLower(filepath.Ext(sf.Path))
		tree, err := p.ParseFile(target.Language, ext, content)
		if err != nil {
			// Close any trees already parsed before returning error
			CloseAll(results)
			return nil, fmt.Errorf("parse %s: %w", sf.RelPath, err)
		}

		results = append(results, &ParsedTreeSitterFile{
			Path:     sf.Path,
			RelPath:  sf.RelPath,
			Tree:     tree,
			Content:  content,
			Language: target.Language,
		})
	}

	return results, nil
}

// CloseAll closes all trees in a slice of ParsedTreeSitterFile.
// Safe to call with nil or empty slice.
func CloseAll(files []*ParsedTreeSitterFile) {
	for _, f := range files {
		if f != nil && f.Tree != nil {
			f.Tree.Close()
		}
	}
}
