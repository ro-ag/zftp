// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestExportedIdentifiersAreDocumented enforces that every exported identifier in
// the zftp package's own source files carries a doc comment, so `go doc` is
// complete for the public surface. It scans the package source directly (skipping
// test files and generated files), and treats a const/var/field as documented if
// it has a doc comment, a trailing line comment, or belongs to a documented
// declaration block — matching what godoc renders.
func TestExportedIdentifiersAreDocumented(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	var undocumented []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if isGeneratedFile(f) {
			continue
		}
		for _, d := range f.Decls {
			switch decl := d.(type) {
			case *ast.FuncDecl:
				if !ast.IsExported(decl.Name.Name) {
					continue
				}
				if decl.Recv != nil {
					// Only methods on exported types are part of the public surface.
					if !ast.IsExported(strings.TrimPrefix(receiverType(decl.Recv), "*")) {
						continue
					}
				}
				if decl.Doc == nil {
					undocumented = append(undocumented, name+": func "+funcLabel(decl))
				}
			case *ast.GenDecl:
				collectUndocumentedGenDecl(name, decl, &undocumented)
			}
		}
	}

	if len(undocumented) > 0 {
		sort.Strings(undocumented)
		t.Fatalf("exported identifiers without a doc comment (%d):\n%s",
			len(undocumented), strings.Join(undocumented, "\n"))
	}
}

func collectUndocumentedGenDecl(file string, decl *ast.GenDecl, out *[]string) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}
			if s.Doc == nil && decl.Doc == nil {
				*out = append(*out, file+": type "+s.Name.Name)
			}
			if st, ok := s.Type.(*ast.StructType); ok && st.Fields != nil {
				for _, fld := range st.Fields.List {
					for _, fn := range fld.Names {
						if ast.IsExported(fn.Name) && fld.Doc == nil && fld.Comment == nil {
							*out = append(*out, file+": field "+s.Name.Name+"."+fn.Name)
						}
					}
				}
			}
		case *ast.ValueSpec:
			exported := false
			for _, n := range s.Names {
				if ast.IsExported(n.Name) {
					exported = true
				}
			}
			if !exported {
				continue
			}
			// A trailing line comment, an own doc comment, or a block-level doc all
			// count — each shows up in godoc.
			if s.Doc == nil && s.Comment == nil && decl.Doc == nil {
				names := make([]string, 0, len(s.Names))
				for _, n := range s.Names {
					names = append(names, n.Name)
				}
				*out = append(*out, file+": value "+strings.Join(names, ","))
			}
		}
	}
}

// isGeneratedFile reports whether f was produced by a code generator (e.g.
// stringer), per the "// Code generated ... DO NOT EDIT." convention.
func isGeneratedFile(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// Code generated ") && strings.HasSuffix(c.Text, "DO NOT EDIT.") {
				return true
			}
		}
	}
	return false
}

func receiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return "*" + id.Name
		}
	}
	return ""
}

func funcLabel(decl *ast.FuncDecl) string {
	if decl.Recv != nil {
		return receiverType(decl.Recv) + "." + decl.Name.Name
	}
	return decl.Name.Name
}
