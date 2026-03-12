package validate

// This package performs semantic validation on the AST.

import (
	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/diagnostics"
	"github.com/MBlore/AuAu/token"
)

type validateContext struct {
	file     *ast.File
	varTypes map[string]*ast.TypeRef
	errors   []error
}

func (ctx *validateContext) addError(nodeMeta ast.NodeMeta, message string) {
	ctx.errors = append(ctx.errors, diagnostics.Error(nodeMeta, message))
}

// Validate takes an AST File and performs semantic validation, returning any errors found.
func Validate(file *ast.File) []error {
	context := &validateContext{
		file:     file,
		varTypes: make(map[string]*ast.TypeRef),
		errors:   []error{},
	}

	ensurePackageDeclared(context)
	ensureUniqueFunctionNames(context)
	ensureUniqueVariableNamesPerBlock(context)
	ensureUniqueExternFuncNames(context)

	inferConstantTypes(context)

	// Check AssignStmt for type correctness, e.g. assigning an int to a string variable should be an error.
	checkAssignStmts(context)

	ensureLoopControlUsedInsideLoop(context)

	return context.errors
}

func ensurePackageDeclared(ctx *validateContext) {
	if ctx.file.PackageName == "" {
		ctx.addError(ast.NodeMeta{}, "package declaration is missing at top of file, e.g. 'package main'")
	}
}

func ensureUniqueFunctionNames(ctx *validateContext) {
	functionNames := make(map[string]bool)
	for _, fn := range ctx.file.Functions {
		if functionNames[fn.Name] {
			ctx.addError(fn.NodeMeta, "duplicate function name: "+fn.Name)
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

// checkBlockForDuplicateVariables checks for duplicate variable names within the same block.
// We also build the varTypes map here, which maps variable names to their declared types,
// for use in type inference later.
func checkBlockForDuplicateVariables(ctx *validateContext, block *ast.BlockStmt) {
	variableNames := make(map[string]bool)

	// We allow shadow variables in nested blocks.
	for _, stmt := range block.Stmts {
		checkStmtForDuplicateVariables(ctx, stmt, variableNames)
	}
}

// checkStmtForDuplicateVariables is used to check for duplicate variable names in statements that aren't blocks, such as else if statements.
func checkStmtForDuplicateVariables(ctx *validateContext, stmt ast.Stmt, variableNames map[string]bool) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		if variableNames[s.Name] {
			ctx.addError(s.NodeMeta, "duplicate variable name in same block: "+s.Name)
		} else {
			variableNames[s.Name] = true
			ctx.varTypes[s.Name] = s.Type
		}

	case *ast.BlockStmt:
		checkBlockForDuplicateVariables(ctx, s)

	case *ast.IfStmt:
		checkBlockForDuplicateVariables(ctx, s.Then)

		if s.Else != nil {
			checkNestedStmtForDuplicateVariables(ctx, s.Else)
		}

	case *ast.WhileStmt:
		checkBlockForDuplicateVariables(ctx, s.Body)

	case *ast.ForStmt:
		// Create a new empty map for loop variables to allow shadowing within the loop.
		loopVariableNames := make(map[string]bool)

		if s.Init != nil {
			checkStmtForDuplicateVariables(ctx, s.Init, loopVariableNames)
		}
		if s.Body != nil {
			checkBlockForDuplicateVariables(ctx, s.Body)
		}
		if s.Post != nil {
			checkStmtForDuplicateVariables(ctx, s.Post, loopVariableNames)
		}
	}
}

// checkNestedStmtForDuplicateVariables handles statements that execute in their own nested scope.
func checkNestedStmtForDuplicateVariables(ctx *validateContext, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		checkBlockForDuplicateVariables(ctx, s)

	case *ast.IfStmt:
		checkBlockForDuplicateVariables(ctx, s.Then)

		if s.Else != nil {
			checkNestedStmtForDuplicateVariables(ctx, s.Else)
		}

	case *ast.WhileStmt:
		checkBlockForDuplicateVariables(ctx, s.Body)

	case *ast.ForStmt:
		loopVariableNames := make(map[string]bool)

		if s.Init != nil {
			checkStmtForDuplicateVariables(ctx, s.Init, loopVariableNames)
		}
		if s.Body != nil {
			checkBlockForDuplicateVariables(ctx, s.Body)
		}
		if s.Post != nil {
			checkStmtForDuplicateVariables(ctx, s.Post, loopVariableNames)
		}

	case *ast.VarDeclStmt:
		ctx.varTypes[s.Name] = s.Type
	}
}

func checkAssignStmts(ctx *validateContext) {
	for _, fn := range ctx.file.Functions {
		checkAssignStmtsInBlock(ctx, fn.Body, make(map[string]*ast.TypeRef))
	}
}

func checkAssignStmtsInBlock(ctx *validateContext, block *ast.BlockStmt, scope map[string]*ast.TypeRef) {
	localScope := cloneTypeScope(scope)

	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *ast.VarDeclStmt:
			localScope[s.Name] = s.Type

		case *ast.AssignStmt:
			varType, ok := localScope[s.Name]
			if !ok {
				ctx.addError(s.NodeMeta, "assignment to undeclared variable: "+s.Name)
				continue
			}

			validateAssignExprType(ctx, localScope, varType, s.Value)

		case *ast.BlockStmt:
			checkAssignStmtsInBlock(ctx, s, localScope)

		case *ast.IfStmt:
			checkAssignStmtsInBlock(ctx, s.Then, localScope)
			if s.Else != nil {
				checkNestedStmtForAssignStmts(ctx, s.Else, localScope)
			}

		case *ast.WhileStmt:
			checkAssignStmtsInBlock(ctx, s.Body, localScope)

		case *ast.ForStmt:
			loopScope := cloneTypeScope(localScope)

			if s.Init != nil {
				checkNestedStmtForAssignStmts(ctx, s.Init, loopScope)
			}
			if s.Body != nil {
				checkAssignStmtsInBlock(ctx, s.Body, loopScope)
			}
			if s.Post != nil {
				checkNestedStmtForAssignStmts(ctx, s.Post, loopScope)
			}
		}
	}
}

func checkNestedStmtForAssignStmts(ctx *validateContext, stmt ast.Stmt, scope map[string]*ast.TypeRef) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		checkAssignStmtsInBlock(ctx, s, scope)

	case *ast.IfStmt:
		checkAssignStmtsInBlock(ctx, s.Then, scope)

		if s.Else != nil {
			checkNestedStmtForAssignStmts(ctx, s.Else, scope)
		}

	case *ast.WhileStmt:
		checkAssignStmtsInBlock(ctx, s.Body, scope)

	case *ast.ForStmt:
		loopScope := cloneTypeScope(scope)
		if s.Init != nil {
			checkNestedStmtForAssignStmts(ctx, s.Init, loopScope)
		}
		if s.Body != nil {
			checkAssignStmtsInBlock(ctx, s.Body, loopScope)
		}
		if s.Post != nil {
			checkNestedStmtForAssignStmts(ctx, s.Post, loopScope)
		}

	case *ast.VarDeclStmt:
		scope[s.Name] = s.Type

	case *ast.AssignStmt:
		varType, ok := scope[s.Name]
		if !ok {
			ctx.addError(s.NodeMeta, "assignment to undeclared variable: "+s.Name)
			return
		}

		validateAssignExprType(ctx, scope, varType, s.Value)
	}
}

func cloneTypeScope(scope map[string]*ast.TypeRef) map[string]*ast.TypeRef {
	clone := make(map[string]*ast.TypeRef, len(scope))

	for name, typ := range scope {
		clone[name] = typ
	}

	return clone
}

func validateAssignExprType(ctx *validateContext, scope map[string]*ast.TypeRef, expectedType *ast.TypeRef, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.StringLiteralExpr:
		if expectedType.Kind != ast.TypeString {
			ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign string to "+ast.TypeToString(expectedType))
		}
	case *ast.BoolLiteralExpr:
		if expectedType.Kind != ast.TypeBool {
			ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign bool to "+ast.TypeToString(expectedType))
		}
	case *ast.IntLiteralExpr:
		if err := literalFitsType(e, false, expectedType); err != nil {
			ctx.addError(e.NodeMeta, err.Error())
			return
		}
		e.InferredType = expectedType
	case *ast.UnaryExpr:
		if e.Op != token.Sub {
			return
		}

		lit, ok := e.Expr.(*ast.IntLiteralExpr)
		if !ok {
			return
		}

		if err := literalFitsType(lit, true, expectedType); err != nil {
			ctx.addError(e.NodeMeta, err.Error())
			return
		}

		lit.InferredType = expectedType
		e.InferredType = expectedType
	case *ast.IdentExpr:
		declType, ok := scope[e.Name]
		if !ok {
			ctx.addError(e.NodeMeta, "undefined variable: "+e.Name)
			return
		}
		if declType.Kind != expectedType.Kind {
			ctx.addError(e.NodeMeta, "type mismatch in assignment to "+e.Name+": cannot assign "+ast.TypeToString(declType)+" to "+ast.TypeToString(expectedType))
		}
	case *ast.BinaryExpr:
		switch e.Op {
		case token.Add, token.Sub, token.Mul, token.Div:
			validateAssignExprType(ctx, scope, expectedType, e.Left)
			validateAssignExprType(ctx, scope, expectedType, e.Right)

			e.InferredType = expectedType

		case token.EqEq, token.NotEq:
			if expectedType.Kind != ast.TypeBool {
				ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign bool to "+ast.TypeToString(expectedType))
			}

			operandType := inferComparisonOperandTypeForScope(scope, e.Left, e.Right)

			validateAssignExprType(ctx, scope, operandType, e.Left)
			validateAssignExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef

		case token.Lt, token.LtEq, token.Gt, token.GtEq:
			if expectedType.Kind != ast.TypeBool {
				ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign bool to "+ast.TypeToString(expectedType))
			}

			operandType := inferComparisonOperandTypeForScope(scope, e.Left, e.Right)

			if !isIntegerType(operandType) {
				ctx.addError(e.NodeMeta, "operator "+ast.TokenTypeToString(e.Op)+" requires integer operands")
				return
			}

			validateAssignExprType(ctx, scope, operandType, e.Left)
			validateAssignExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		}
	}
}

func inferComparisonOperandTypeForScope(scope map[string]*ast.TypeRef, left, right ast.Expr) *ast.TypeRef {
	if t := exprKnownTypeForScope(scope, left); t != nil {
		return t
	}

	if t := exprKnownTypeForScope(scope, right); t != nil {
		return t
	}

	return ast.TypeIntRef
}

func exprKnownTypeForScope(scope map[string]*ast.TypeRef, expr ast.Expr) *ast.TypeRef {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return scope[e.Name]
	case *ast.IntLiteralExpr:
		return e.InferredType
	case *ast.UnaryExpr:
		return e.InferredType
	case *ast.BinaryExpr:
		return e.InferredType
	case *ast.BoolLiteralExpr:
		return ast.TypeBoolRef
	case *ast.StringLiteralExpr:
		return ast.TypeStringRef
	default:
		return nil
	}
}
