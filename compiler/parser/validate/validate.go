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
	structs  map[string]*ast.StructDecl
	funcs    map[string]funcSig
	errors   []error
}

type funcSig struct {
	params     []*ast.TypeRef
	returnType *ast.TypeRef
}

func (ctx *validateContext) addError(nodeMeta ast.NodeMeta, message string) {
	ctx.errors = append(ctx.errors, diagnostics.Error(nodeMeta, message))
}

// Validate takes an AST File and performs semantic validation, returning any errors found.
func Validate(file *ast.File) []error {
	context := &validateContext{
		file:     file,
		varTypes: make(map[string]*ast.TypeRef),
		structs:  make(map[string]*ast.StructDecl),
		funcs:    make(map[string]funcSig),
		errors:   []error{},
	}

	for _, s := range file.Structs {
		if _, exists := context.structs[s.Name]; !exists {
			context.structs[s.Name] = s
		}
	}

	for _, fn := range file.Functions {
		for _, param := range fn.Params {
			context.varTypes[param.Name] = param.Type
		}
	}

	for _, ext := range file.Externs {
		params := make([]*ast.TypeRef, 0, len(ext.Params))
		for _, param := range ext.Params {
			params = append(params, param.Type)
		}

		context.funcs[ext.Name] = funcSig{
			params:     params,
			returnType: ext.ReturnType,
		}
	}

	for _, fn := range file.Functions {
		params := make([]*ast.TypeRef, 0, len(fn.Params))
		for _, param := range fn.Params {
			params = append(params, param.Type)
		}

		context.funcs[fn.Name] = funcSig{
			params:     params,
			returnType: fn.ReturnType,
		}
	}

	ensurePackageDeclared(context)
	ensureUniqueFunctionNames(context)
	ensureUniqueStructNames(context)
	ensureUniqueVariableNamesPerBlock(context)
	ensureUniqueExternFuncNames(context)
	ensureKnownCustomTypes(context)
	ensureNoVoidVariables(context)
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
		scope := make(map[string]*ast.TypeRef, len(fn.Params))
		for _, param := range fn.Params {
			scope[param.Name] = param.Type
		}

		checkAssignStmtsInBlock(ctx, fn.Body, scope)
	}
}

func checkAssignStmtsInBlock(ctx *validateContext, block *ast.BlockStmt, scope map[string]*ast.TypeRef) {
	localScope := cloneTypeScope(scope)

	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *ast.VarDeclStmt:
			localScope[s.Name] = s.Type

		case *ast.AssignStmt:
			targetType := resolveExprType(ctx, localScope, s.Target)
			if targetType == nil {
				continue
			}

			validateAssignExprType(ctx, localScope, targetType, s.Value)

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
		targetType := resolveExprType(ctx, scope, s.Target)
		if targetType == nil {
			return
		}

		validateAssignExprType(ctx, scope, targetType, s.Value)
	}
}

func assignmentTargetBaseName(target ast.Expr) (string, bool) {
	switch t := target.(type) {
	case *ast.IdentExpr:
		return t.Name, true
	case *ast.FieldAccessExpr:
		return assignmentTargetBaseName(t.Base)
	default:
		return "", false
	}
}

func resolveExprType(ctx *validateContext, scope map[string]*ast.TypeRef, expr ast.Expr) *ast.TypeRef {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		declType, ok := lookupType(scope, ctx, e.Name)
		if !ok {
			ctx.addError(e.NodeMeta, "undefined variable: "+e.Name)
			return nil
		}
		return declType
	case *ast.FieldAccessExpr:
		baseType := resolveExprType(ctx, scope, e.Base)
		if baseType == nil {
			return nil
		}

		if baseType.Kind != ast.TypeCustom {
			ctx.addError(e.NodeMeta, "field access requires struct type, got "+ast.TypeToString(baseType))
			return nil
		}

		structDecl, ok := ctx.structs[baseType.Name]
		if !ok {
			ctx.addError(e.NodeMeta, "unknown struct type: "+baseType.Name)
			return nil
		}

		for _, field := range structDecl.Fields {
			if field.Name == e.Field {
				return field.Type
			}
		}

		ctx.addError(e.NodeMeta, "struct "+baseType.Name+" has no field "+e.Field)
		return nil
	case *ast.CallExpr:
		return validateCallExpr(ctx, scope, e)

	default:
		ctx.addError(ast.NodeMeta{}, "invalid assignment target")
		return nil
	}
}

func sameType(a *ast.TypeRef, b *ast.TypeRef) bool {
	if a == nil || b == nil {
		return a == b
	}

	if a.Kind != b.Kind {
		return false
	}

	if a.Kind == ast.TypeCustom {
		return a.Name == b.Name
	}

	return true
}

func cloneTypeScope(scope map[string]*ast.TypeRef) map[string]*ast.TypeRef {
	clone := make(map[string]*ast.TypeRef, len(scope))

	for name, typ := range scope {
		clone[name] = typ
	}

	return clone
}

func lookupType(scope map[string]*ast.TypeRef, ctx *validateContext, name string) (*ast.TypeRef, bool) {
	if t, ok := scope[name]; ok {
		return t, true
	}

	t, ok := ctx.varTypes[name]
	return t, ok
}

func validateCallExpr(ctx *validateContext, scope map[string]*ast.TypeRef, call *ast.CallExpr) *ast.TypeRef {
	sig, ok := ctx.funcs[call.FuncName]
	if !ok {
		ctx.addError(call.NodeMeta, "unknown function: "+call.FuncName)
		return nil
	}

	if len(sig.params) != len(call.Args) {
		ctx.addError(call.NodeMeta, "function "+call.FuncName+" expects "+itoa(len(sig.params))+" arguments, got "+itoa(len(call.Args)))
		call.InferredType = sig.returnType
		return sig.returnType
	}

	for i, arg := range call.Args {
		validateExprType(ctx, scope, sig.params[i], arg)
	}

	call.InferredType = sig.returnType
	return sig.returnType
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}

	return string(buf[i:])
}

func validateAssignExprType(ctx *validateContext, scope map[string]*ast.TypeRef, expectedType *ast.TypeRef, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.CallExpr:
		retType := validateCallExpr(ctx, scope, e)
		if retType == nil {
			return
		}

		if !sameType(retType, expectedType) {
			ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign "+ast.TypeToString(retType)+" to "+ast.TypeToString(expectedType))
		}
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
	case *ast.FloatLiteralExpr:
		if err := floatLiteralFitsType(e, expectedType); err != nil {
			ctx.addError(e.NodeMeta, err.Error())
			return
		}
		e.InferredType = expectedType
	case *ast.UnaryExpr:
		if e.Op != token.Sub {
			return
		}

		// Check expression is either a int or float literal, and if so, validate the literal fits the expected type.
		switch lit := e.Expr.(type) {
		case *ast.IntLiteralExpr:
			if err := literalFitsType(lit, true, expectedType); err != nil {
				ctx.addError(e.NodeMeta, err.Error())
				return
			}

			lit.InferredType = expectedType
		case *ast.FloatLiteralExpr:
			if err := floatLiteralFitsType(lit, expectedType); err != nil {
				ctx.addError(e.NodeMeta, err.Error())
				return
			}

			lit.InferredType = expectedType

		default:
			ctx.addError(e.NodeMeta, "unary '-' operator can only be applied to int or float literals")
			return
		}

		e.InferredType = expectedType
	case *ast.IdentExpr:
		declType := resolveExprType(ctx, scope, e)
		if declType == nil {
			return
		}
		if !sameType(declType, expectedType) {
			ctx.addError(e.NodeMeta, "type mismatch in assignment to "+e.Name+": cannot assign "+ast.TypeToString(declType)+" to "+ast.TypeToString(expectedType))
		}
	case *ast.FieldAccessExpr:
		fieldType := resolveExprType(ctx, scope, e)
		if fieldType == nil {
			return
		}
		if !sameType(fieldType, expectedType) {
			ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign "+ast.TypeToString(fieldType)+" to "+ast.TypeToString(expectedType))
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

			operandType := inferComparisonOperandTypeForScope(ctx, scope, e.Left, e.Right)

			validateAssignExprType(ctx, scope, operandType, e.Left)
			validateAssignExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef

		case token.Lt, token.LtEq, token.Gt, token.GtEq:
			if expectedType.Kind != ast.TypeBool {
				ctx.addError(e.NodeMeta, "type mismatch in assignment: cannot assign bool to "+ast.TypeToString(expectedType))
			}

			operandType := inferComparisonOperandTypeForScope(ctx, scope, e.Left, e.Right)

			if !isIntegerType(operandType) && !isFloatType(operandType) {
				ctx.addError(e.NodeMeta, "operator "+ast.TokenTypeToString(e.Op)+" requires integer or float operands")
				return
			}

			validateAssignExprType(ctx, scope, operandType, e.Left)
			validateAssignExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		}
	}
}

func inferComparisonOperandTypeForScope(ctx *validateContext, scope map[string]*ast.TypeRef, left, right ast.Expr) *ast.TypeRef {
	if t := exprKnownTypeForScope(ctx, scope, left); t != nil {
		return t
	}

	if t := exprKnownTypeForScope(ctx, scope, right); t != nil {
		return t
	}

	if _, ok := left.(*ast.FloatLiteralExpr); ok {
		return ast.TypeFloatRef
	}
	if _, ok := right.(*ast.FloatLiteralExpr); ok {
		return ast.TypeFloatRef
	}

	return ast.TypeIntRef
}

// exprKnownTypeForScope checks if the expression is a literal or identifier with a known type in the current scope, and returns that type if so.
// This is used to help infer the type of comparison expressions when validating assignments, e.g. in 'x = 5 < 3.2'.
func exprKnownTypeForScope(ctx *validateContext, scope map[string]*ast.TypeRef, expr ast.Expr) *ast.TypeRef {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return scope[e.Name]
	case *ast.FieldAccessExpr:
		return resolveExprType(ctx, scope, e)
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
	case *ast.FloatLiteralExpr:
		return e.InferredType
	case *ast.CallExpr:
		return e.InferredType
	default:
		return nil
	}
}
