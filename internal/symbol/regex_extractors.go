package symbol

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

// --- TypeScript / JavaScript ---

var reTSArrowConst = regexp.MustCompile(`^(?:export\s+)?const\s+(\w+)\s*(?::\s*\S+\s*)?=\s*(?:async\s+)?\(?`)

func extractTypeScript(src []byte) ([]Symbol, error) {
	return extractWithScanner(src, func(line string, lineNum int) *Symbol {
		line = strings.TrimSpace(line)
		if sym := matchTSLine(line, lineNum); sym != nil {
			return sym
		}
		return nil
	}), nil
}

func extractJavaScript(src []byte) ([]Symbol, error) {
	return extractTypeScript(src) // same patterns
}

func matchTSLine(line string, lineNum int) *Symbol {
	// function declarations
	if m := regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?(?:async\s+)?function\*?\s+(\w+)`).FindStringSubmatch(line); m != nil {
		return &Symbol{Name: m[1], Kind: KindFunction, Line: lineNum}
	}
	// class
	if m := regexp.MustCompile(`^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`).FindStringSubmatch(line); m != nil {
		return &Symbol{Name: m[1], Kind: KindClass, Line: lineNum}
	}
	// interface
	if m := regexp.MustCompile(`^(?:export\s+)?interface\s+(\w+)`).FindStringSubmatch(line); m != nil {
		return &Symbol{Name: m[1], Kind: KindInterface, Line: lineNum}
	}
	// type alias
	if m := regexp.MustCompile(`^(?:export\s+)?type\s+(\w+)\s*[=<]`).FindStringSubmatch(line); m != nil {
		return &Symbol{Name: m[1], Kind: KindType, Line: lineNum}
	}
	// enum
	if m := regexp.MustCompile(`^(?:export\s+)?(?:const\s+)?enum\s+(\w+)`).FindStringSubmatch(line); m != nil {
		return &Symbol{Name: m[1], Kind: KindEnum, Line: lineNum}
	}
	// const arrow functions: export const foo = (...) =>
	if m := reTSArrowConst.FindStringSubmatch(line); m != nil {
		if strings.Contains(line, "=>") || strings.HasSuffix(strings.TrimSpace(line), "{") {
			return &Symbol{Name: m[1], Kind: KindFunction, Line: lineNum}
		}
	}
	return nil
}

// --- Python ---

var (
	rePyFunc  = regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`)
	rePyClass = regexp.MustCompile(`^class\s+(\w+)`)
)

func extractPython(src []byte) ([]Symbol, error) {
	return extractWithScanner(src, func(line string, lineNum int) *Symbol {
		if m := rePyFunc.FindStringSubmatch(line); m != nil {
			if !strings.HasPrefix(m[1], "_") { // skip private
				return &Symbol{Name: m[1], Kind: KindFunction, Line: lineNum}
			}
		}
		if m := rePyClass.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindClass, Line: lineNum}
		}
		return nil
	}), nil
}

// --- Rust ---

var (
	reRustFn     = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?(?:async\s+)?fn\s+(\w+)`)
	reRustStruct = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?struct\s+(\w+)`)
	reRustEnum   = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?enum\s+(\w+)`)
	reRustTrait  = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?trait\s+(\w+)`)
	reRustType   = regexp.MustCompile(`^(?:pub(?:\([^)]*\))?\s+)?type\s+(\w+)`)
)

func extractRust(src []byte) ([]Symbol, error) {
	return extractWithScanner(src, func(line string, lineNum int) *Symbol {
		line = strings.TrimSpace(line)
		if m := reRustFn.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindFunction, Line: lineNum}
		}
		if m := reRustStruct.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindStruct, Line: lineNum}
		}
		if m := reRustEnum.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindEnum, Line: lineNum}
		}
		if m := reRustTrait.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindTrait, Line: lineNum}
		}
		if m := reRustType.FindStringSubmatch(line); m != nil {
			return &Symbol{Name: m[1], Kind: KindType, Line: lineNum}
		}
		return nil
	}), nil
}

// --- Helper ---

func extractWithScanner(src []byte, match func(line string, lineNum int) *Symbol) []Symbol {
	var symbols []Symbol
	scanner := bufio.NewScanner(bytes.NewReader(src))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if sym := match(scanner.Text(), lineNum); sym != nil {
			symbols = append(symbols, *sym)
		}
	}
	return symbols
}

