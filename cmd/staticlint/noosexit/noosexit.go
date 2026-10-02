// Package noosexit defines an analyzer that forbids a direct call to
// os.Exit inside func main of package main.
package noosexit

import (
	"go/ast"
	"go/types"
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

		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				// Calls inside func main are the only ones we care about,
				// so skip the bodies of every other function and method.
				return n.Recv == nil && n.Name.Name == "main"
			case *ast.GenDecl:
				// Package-level var/const initializers are not part of main.
				return false
			case *ast.CallExpr:
				if isOSExit(pass, n) {
					pass.Reportf(n.Pos(), "direct call to os.Exit in main function of package main is forbidden")
				}
			}
			return true
		})
	}

	return nil, nil
}

// isOSExit reports whether call is a call to the Exit function of the
// standard os package, regardless of the name os was imported under.
func isOSExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Exit" {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
	return ok && pkgName.Imported().Path() == "os"
}
