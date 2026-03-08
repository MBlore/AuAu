package validate

// This package performs semantic validation on the AST.

import (
	"errors"

	"github.com/MBlore/AuAu/ast"
)

type validateContext struct {
	file   *ast.File
	errors []error
}

// Validate takes an AST File and performs semantic validation, returning any errors found.
func Validate(file *ast.File) []error {
	context := &validateContext{
		file: file,
	}

	ensurePackageDeclared(context)
	ensureUniqueFunctionNames(context)
	ensureUniqueVariableNamesPerBlock(context)

	return context.errors
}

func ensurePackageDeclared(ctx *validateContext) {
	if ctx.file.PackageName == "" {
		ctx.errors = append(ctx.errors, errors.New("package declaration is missing at top of file, e.g. 'package main'"))
	}
}

func ensureUniqueFunctionNames(ctx *validateContext) {
	functionNames := make(map[string]bool)
	for _, fn := range ctx.file.Functions {
		if functionNames[fn.Name] {
			ctx.errors = append(ctx.errors, errors.New("duplicate function name: "+fn.Name))
		} else {
			functionNames[fn.Name] = true
		}
	}
}

func ensureUniqueVariableNamesPerBlock(ctx *validateContext) {
	for _, fn := range ctx.file.Functions {
		checkBlockForDuplicateVariables(ctx, fn.Body)
	}
}

func checkBlockForDuplicateVariables(ctx *validateContext, block *ast.BlockStmt) {
	variableNames := make(map[string]bool)

	// We allow shadow variables in nested blocks.

	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *ast.VarDeclStmt:
			if variableNames[s.Name] {
				ctx.errors = append(ctx.errors, errors.New("duplicate variable name in same block: "+s.Name))
			} else {
				variableNames[s.Name] = true
			}
		case *ast.BlockStmt:
			// Recurse into nested blocks.
			checkBlockForDuplicateVariables(ctx, s)
		}
	}
}
