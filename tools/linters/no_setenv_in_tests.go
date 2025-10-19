package main

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `no_setenv_in_tests: prevent os.Setenv and t.Setenv usage in test files

This linter enforces the "Config-First" dependency injection pattern by preventing
the use of os.Setenv and t.Setenv in test files. These functions modify global state
and can cause data races in parallel test execution.

Instead of using os.Setenv or t.Setenv, tests should:
1. Get base config from testutil.Setup(t)
2. Create a local copy of the config
3. Modify the local copy as needed
4. Pass the modified config to constructors

This ensures parallel-safe test execution and follows the Config-First pattern.
`

var Analyzer = &analysis.Analyzer{
	Name:     "no_setenv_in_tests",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if !isTestFile(pass.Fset, pass.Files) {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		
		fun, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		pkg, ok := fun.X.(*ast.Ident)
		if !ok {
			return
		}

		pkgName := pkg.Name
		funcName := fun.Sel.Name

		if pkgName == "os" && funcName == "Setenv" {
			pass.Reportf(call.Pos(), 
				"os.Setenv is forbidden in test files. Use Config-First pattern: "+
				"get base config from testutil.Setup(t), create local copy, modify it, "+
				"and pass to constructors instead of modifying global state.")
		}

		if funcName == "Setenv" {
			if isTestingT(pass, fun.X) {
				pass.Reportf(call.Pos(),
					"t.Setenv is forbidden in test files. Use Config-First pattern: "+
					"get base config from testutil.Setup(t), create local copy, modify it, "+
					"and pass to constructors instead of modifying global state.")
			}
		}
	})

	return nil, nil
}

func isTestFile(fset *token.FileSet, files []*ast.File) bool {
	for _, file := range files {
		filename := fset.Position(file.Package).Filename
		if strings.HasSuffix(filename, "_test.go") {
			return true
		}
	}
	return false
}

func isTestingT(pass *analysis.Pass, expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj != nil {
			return ident.Name == "t" || ident.Name == "testingT"
		}
	}

	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "test" && sel.Sel.Name == "t"
		}
	}

	return false
}
