package symbol

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func extractGo(path string, src []byte) ([]Symbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		// Return what we can even if there are parse errors.
		if f == nil {
			return nil, nil
		}
	}

	var symbols []Symbol
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := Symbol{
				Line: fset.Position(d.Pos()).Line,
			}
			if d.Recv != nil {
				sym.Kind = KindMethod
			} else {
				sym.Kind = KindFunction
			}
			sym.Name = d.Name.Name
			if d.Doc != nil {
				sym.Comment = strings.TrimSpace(d.Doc.Text())
			}
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					sym := Symbol{
						Name: s.Name.Name,
						Line: fset.Position(s.Pos()).Line,
					}
					switch s.Type.(type) {
					case *ast.InterfaceType:
						sym.Kind = KindInterface
					case *ast.StructType:
						sym.Kind = KindStruct
					default:
						sym.Kind = KindType
					}
					if d.Doc != nil {
						sym.Comment = strings.TrimSpace(d.Doc.Text())
					} else if s.Comment != nil {
						sym.Comment = strings.TrimSpace(s.Comment.Text())
					}
					symbols = append(symbols, sym)

				case *ast.ValueSpec:
					kind := KindVar
					if d.Tok == token.CONST {
						kind = KindConst
					}
					for _, name := range s.Names {
						if !ast.IsExported(name.Name) {
							continue
						}
						sym := Symbol{
							Name: name.Name,
							Kind: kind,
							Line: fset.Position(name.Pos()).Line,
						}
						if d.Doc != nil {
							sym.Comment = strings.TrimSpace(d.Doc.Text())
						}
						symbols = append(symbols, sym)
					}
				}
			}
		}
	}

	return symbols, nil
}
