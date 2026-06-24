// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestExportedIdentifiersAreDocumented enforces that every exported identifier in
// the module's public packages carries a doc comment, so `go doc` is complete for
// the public surface. It scans the root package plus the exported subpackages
// (hfs, eol) directly (skipping test files and generated files), and treats a
// const/var/field as documented if it has a doc comment, a trailing line comment,
// or belongs to a documented declaration block — matching what godoc renders.
// Findings are reported as "pkg/ident" (the root package uses "." as its label).
func TestExportedIdentifiersAreDocumented(t *testing.T) {
	// Directories of the module's exported packages, relative to the repo root.
	// The doc gate must hold for every package a consumer can import.
	dirs := []string{".", "hfs", "eol"}

	var undocumented []string
	for _, dir := range dirs {
		undocumented = append(undocumented, undocumentedExportedIdents(t, dir)...)
	}

	if len(undocumented) > 0 {
		sort.Strings(undocumented)
		t.Fatalf("exported identifiers without a doc comment (%d):\n%s",
			len(undocumented), strings.Join(undocumented, "\n"))
	}
}

// undocumentedExportedIdents parses every non-test, non-generated .go file in dir
// and returns the exported identifiers that lack documentation. Each finding is
// prefixed with dir (the root package "." is reported as "."), e.g.
// "hfs: type Dataset" or ".: func Dial".
func undocumentedExportedIdents(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir %q: %v", dir, err)
	}

	var undocumented []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if isGeneratedFile(f) {
			continue
		}
		label := dir + ": "
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
					undocumented = append(undocumented, label+"func "+funcLabel(decl))
				}
			case *ast.GenDecl:
				collectUndocumentedGenDecl(label, decl, &undocumented)
			}
		}
	}
	return undocumented
}

// collectUndocumentedGenDecl appends undocumented exported types, struct fields,
// and consts/vars from decl to out. label is the per-package prefix (e.g. "hfs: ")
// already including its trailing separator.
func collectUndocumentedGenDecl(label string, decl *ast.GenDecl, out *[]string) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}
			if s.Doc == nil && decl.Doc == nil {
				*out = append(*out, label+"type "+s.Name.Name)
			}
			if st, ok := s.Type.(*ast.StructType); ok && st.Fields != nil {
				for _, fld := range st.Fields.List {
					for _, fn := range fld.Names {
						if ast.IsExported(fn.Name) && fld.Doc == nil && fld.Comment == nil {
							*out = append(*out, label+"field "+s.Name.Name+"."+fn.Name)
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
				*out = append(*out, label+"value "+strings.Join(names, ","))
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
