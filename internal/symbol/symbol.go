// Package symbol extracts top-level code symbols from source files.
// It supports Go (via go/ast), TypeScript/JavaScript, Python, and Rust
// (via regular expressions).
package symbol

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind describes what kind of symbol was extracted.
type Kind string

const (
	KindFunction  Kind = "function"
	KindMethod    Kind = "method"
	KindType      Kind = "type"
	KindInterface Kind = "interface"
	KindClass     Kind = "class"
	KindStruct    Kind = "struct"
	KindEnum      Kind = "enum"
	KindTrait     Kind = "trait"
	KindConst     Kind = "const"
	KindVar       Kind = "var"
)

// Symbol represents a single named symbol extracted from a source file.
type Symbol struct {
	Name    string
	Kind    Kind
	Line    int    // 1-based line number
	Comment string // leading doc comment or "" if none
}

// ExtractFile reads the file at path and returns its top-level symbols.
// Returns nil, nil for unsupported file types.
func ExtractFile(path string) ([]Symbol, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return Extract(path, src)
}

// Extract extracts symbols from src using the appropriate strategy for the
// file extension inferred from path.
func Extract(path string, src []byte) ([]Symbol, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return extractGo(path, src)
	case ".ts", ".tsx":
		return extractTypeScript(src)
	case ".js", ".jsx", ".mjs":
		return extractJavaScript(src)
	case ".py":
		return extractPython(src)
	case ".rs":
		return extractRust(src)
	default:
		return nil, nil
	}
}

// MemoryContent returns a short, human-readable description of a symbol
// suitable for use as memory content.
func MemoryContent(sym Symbol, filePath string) string {
	if sym.Comment != "" {
		return fmt.Sprintf("%s `%s` in %s — %s", capitalize(string(sym.Kind)), sym.Name, filePath, sym.Comment)
	}
	return fmt.Sprintf("%s `%s` defined in %s", capitalize(string(sym.Kind)), sym.Name, filePath)
}

// Tags returns tags for a symbol memory: ["symbol", "<kind>", "<language>"].
func Tags(sym Symbol, path string) []string {
	ext := strings.ToLower(filepath.Ext(path))
	lang := extToLang(ext)
	tags := []string{"symbol", string(sym.Kind)}
	if lang != "" {
		tags = append(tags, lang)
	}
	return tags
}

func extToLang(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
