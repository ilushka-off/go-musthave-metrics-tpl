package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// resetMarker is the comment that must appear directly above a struct
// declaration for the generator to produce a Reset method for it.
const resetMarker = "generate:reset"

// zeroValues maps the name of a predeclared basic type to its zero-value
// literal.
var zeroValues = map[string]string{
	"bool":       "false",
	"string":     `""`,
	"int":        "0",
	"int8":       "0",
	"int16":      "0",
	"int32":      "0",
	"int64":      "0",
	"uint":       "0",
	"uint8":      "0",
	"uint16":     "0",
	"uint32":     "0",
	"uint64":     "0",
	"uintptr":    "0",
	"byte":       "0",
	"rune":       "0",
	"float32":    "0",
	"float64":    "0",
	"complex64":  "0",
	"complex128": "0",
}

type resetStruct struct {
	name  string
	stmts []string
}

type resetPackage struct {
	dir     string
	name    string
	structs []resetStruct
}

// run scans every Go package under root and (re)generates a reset.gen.go
// file for each package that has at least one struct annotated with
// "generate:reset".
func run(root string) error {
	packages, err := collectPackages(root)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		if err := writePackage(pkg); err != nil {
			return fmt.Errorf("%s: %w", pkg.dir, err)
		}
	}

	return nil
}

// collectPackages walks root and groups every annotated struct it finds by
// the directory (== package) it was declared in.
func collectPackages(root string) ([]*resetPackage, error) {
	byDir := make(map[string]*resetPackage)
	var order []string

	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if path != root && (name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || filepath.Base(path) == "reset.gen.go" {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		structs := collectStructs(file)
		if len(structs) == 0 {
			return nil
		}

		dir := filepath.Dir(path)
		pkg, ok := byDir[dir]
		if !ok {
			pkg = &resetPackage{dir: dir, name: file.Name.Name}
			byDir[dir] = pkg
			order = append(order, dir)
		}
		pkg.structs = append(pkg.structs, structs...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]*resetPackage, 0, len(order))
	for _, dir := range order {
		result = append(result, byDir[dir])
	}
	return result, nil
}

// collectStructs returns every struct declared in file that carries the
// resetMarker comment.
func collectStructs(file *ast.File) []resetStruct {
	var structs []resetStruct

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.TypeParams != nil {
				continue
			}

			var doc strings.Builder
			if gd.Doc != nil {
				doc.WriteString(gd.Doc.Text())
			}
			if ts.Doc != nil {
				doc.WriteString(ts.Doc.Text())
			}
			if !strings.Contains(doc.String(), resetMarker) {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structs = append(structs, buildStruct(ts.Name.Name, st))
		}
	}

	return structs
}

// buildStruct computes the Reset method body (as a list of statements) for
// every field of a struct named name.
func buildStruct(name string, st *ast.StructType) resetStruct {
	recv := receiverName(name)
	rs := resetStruct{name: name}

	if st.Fields == nil {
		return rs
	}

	for _, field := range st.Fields.List {
		names := field.Names
		if len(names) == 0 {
			// Embedded field: its accessor is the type's own name.
			fieldName := embeddedFieldName(field.Type)
			if fieldName == "" {
				continue
			}
			names = []*ast.Ident{{Name: fieldName}}
		}

		for _, n := range names {
			if n.Name == "_" {
				continue
			}
			access := recv + "." + n.Name
			if stmt := resetStmt(access, field.Type); stmt != "" {
				rs.stmts = append(rs.stmts, stmt)
			}
		}
	}

	return rs
}

// receiverName derives a short receiver identifier from a struct name, e.g.
// "Foo" -> "f".
func receiverName(structName string) string {
	if structName == "" {
		return "r"
	}
	return strings.ToLower(structName[:1])
}

// embeddedFieldName returns the identifier Go generates for an embedded
// field of type expr, e.g. "*pkg.Foo" -> "Foo".
func embeddedFieldName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(t.X)
	default:
		return ""
	}
}

// resetStmt returns the Go statement that resets the field reachable via
// access, based on its syntactic type. Types the generator doesn't
// recognize as primitives, slices or maps fall back to a runtime check for
// a Reset() method, so nested structs (by value or by pointer) are reset
// transitively without the generator needing full type information.
func resetStmt(access string, typ ast.Expr) string {
	switch t := typ.(type) {
	case *ast.Ident:
		if zero, ok := zeroValues[t.Name]; ok {
			return fmt.Sprintf("%s = %s", access, zero)
		}
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			if zero, ok := zeroValues[id.Name]; ok {
				return fmt.Sprintf("if %s != nil {\n*%s = %s\n}", access, access, zero)
			}
		}
		return fmt.Sprintf("if %s != nil {\n%s\n}", access, resetterCheck(access, true))
	case *ast.ArrayType:
		if t.Len == nil {
			return fmt.Sprintf("%s = %s[:0]", access, access)
		}
		return ""
	case *ast.MapType:
		return fmt.Sprintf("clear(%s)", access)
	}

	return resetterCheck(access, false)
}

// resetterCheck returns a statement that, at runtime, checks whether the
// value reachable via access implements interface{ Reset() } and calls it
// if so. isPointer indicates whether access is already a pointer (in which
// case it's asserted directly) or a value (in which case its address is
// taken, so a Reset method with a pointer receiver is still found).
func resetterCheck(access string, isPointer bool) string {
	expr := access
	if !isPointer {
		expr = "&" + access
	}
	return fmt.Sprintf("if resetter, ok := interface{}(%s).(interface{ Reset() }); ok {\nresetter.Reset()\n}", expr)
}

// writePackage renders pkg's reset.gen.go and writes it to pkg.dir.
func writePackage(pkg *resetPackage) error {
	var buf bytes.Buffer

	buf.WriteString("// Code generated by cmd/reset. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", pkg.name)

	for _, st := range pkg.structs {
		recv := receiverName(st.name)
		fmt.Fprintf(&buf, "func (%s *%s) Reset() {\n", recv, st.name)
		fmt.Fprintf(&buf, "if %s == nil {\nreturn\n}\n\n", recv)
		for _, stmt := range st.stmts {
			buf.WriteString(stmt)
			buf.WriteString("\n")
		}
		buf.WriteString("}\n\n")
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format generated code: %w", err)
	}

	return os.WriteFile(filepath.Join(pkg.dir, "reset.gen.go"), formatted, 0o644)
}
