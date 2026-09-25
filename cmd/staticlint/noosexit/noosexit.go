// Package noosexit defines an analyzer that forbids a direct call to
// os.Exit inside func main of package main.
package noosexit

import (
	"go/ast"
	"regexp"

	"golang.org/x/tools/go/analysis"
)

// generatedCodeRe matches the standard "generated code" marker comment
// described at https://go.dev/s/generatedcode, e.g. the wrapper main that
// 'go test' synthesizes to call os.Exit(m.Run()) for a test binary.
var generatedCodeRe = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGenerated(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}
		for _, c := range group.List {
			if generatedCodeRe.MatchString(c.Text) {
				return true
			}
		}
	}
	return false
}

// Analyzer reports direct calls to os.Exit made from inside func main of
// package main. Calling os.Exit there skips any deferred cleanup in main
// and makes that code path harder to test; callers should let main return
// normally instead.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "check for a direct call to os.Exit in func main of package main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if isGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Exit" {
					return true
				}

				ident, ok := sel.X.(*ast.Ident)
				if !ok || ident.Name != "os" {
					return true
				}

				pass.Reportf(call.Pos(), "direct call to os.Exit in main function of package main is forbidden")
				return true
			})
		}
	}

	return nil, nil
}
