package validate

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

func inferConstantTypes(ctx *validateContext) {
	// From here, we need to walk all the function blocks, all the statements in each block,
	// and all the expressions and literals in each statement, and infer types for any literals we find.
	for _, fn := range ctx.file.Functions {
		inferTypesInBlock(ctx, fn.Body, make(map[string]*ast.TypeRef))
	}
}

func inferTypesInBlock(ctx *validateContext, block *ast.BlockStmt, scope map[string]*ast.TypeRef) {
	localScope := cloneTypeScope(scope)

	for _, stmt := range block.Stmts {
		inferTypesInStmt(ctx, stmt, localScope)
	}
}

// inferTypesInStmt is used to infer types in statements that aren't blocks, such as else if statements.
func inferTypesInStmt(ctx *validateContext, stmt ast.Stmt, scope map[string]*ast.TypeRef) {
	switch s := stmt.(type) {
	case *ast.CallStmt:
		for _, arg := range s.Args {
			inferExprDefaultType(ctx, scope, arg)
		}
	case *ast.VarDeclStmt:
		if s.Init != nil {
			validateExprType(ctx, scope, s.Type, s.Init)
		}
		scope[s.Name] = s.Type
	case *ast.IfStmt:
		validateExprType(ctx, scope, ast.TypeBoolRef, s.Cond)
		inferTypesInBlock(ctx, s.Then, scope)
		if s.Else != nil {
			inferTypesInStmt(ctx, s.Else, cloneTypeScope(scope))
		}
	case *ast.BlockStmt:
		inferTypesInBlock(ctx, s, scope)
	case *ast.WhileStmt:
		validateExprType(ctx, scope, ast.TypeBoolRef, s.Cond)
		inferTypesInBlock(ctx, s.Body, scope)
	case *ast.ForStmt:
		loopScope := cloneTypeScope(scope)

		if s.Init != nil {
			inferTypesInStmt(ctx, s.Init, loopScope)
		}
		if s.Cond != nil {
			validateExprType(ctx, loopScope, ast.TypeBoolRef, s.Cond)
		}
		if s.Body != nil {
			inferTypesInBlock(ctx, s.Body, loopScope)
		}
		if s.Post != nil {
			inferTypesInStmt(ctx, s.Post, loopScope)
		}
	}
}

// validateExprType ensures that the expression tree matches the expected type.
func validateExprType(ctx *validateContext, scope map[string]*ast.TypeRef, expectedType *ast.TypeRef, expr ast.Expr) {
	// We know from the declaration type what the expression should be,
	// but have to verify the literal will fit in that type and set the literal's type accordingly.

	// Now we look for IntLiteralExpr in the expression graph and set their type to the inferred type.
	switch e := expr.(type) {
	case *ast.StringLiteralExpr:
		if expectedType.Kind != ast.TypeString {
			ctx.addError(e.NodeMeta, "type mismatch: expected "+ast.TypeKindToString(expectedType.Kind)+", got string")
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
			ctx.addError(e.NodeMeta, "unary operator '-' requires integer or float literal operand")
			return
		}

		e.InferredType = expectedType
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EqEq, token.NotEq:
			if expectedType.Kind != ast.TypeBool {
				ctx.addError(e.NodeMeta, "type mismatch: expected "+ast.TypeKindToString(expectedType.Kind)+", got bool")
			}

			operandType := inferComparisonOperandType(scope, e.Left, e.Right)

			validateExprType(ctx, scope, operandType, e.Left)
			validateExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		case token.Add, token.Sub, token.Mul, token.Div:
			// For binary expressions, we need to validate both sides.
			validateExprType(ctx, scope, expectedType, e.Left)
			validateExprType(ctx, scope, expectedType, e.Right)

			e.InferredType = expectedType
		case token.Lt, token.LtEq, token.Gt, token.GtEq:
			// Check expected type is valid for comparison operators.
			if expectedType.Kind != ast.TypeBool {
				ctx.addError(e.NodeMeta, "type mismatch: expected "+ast.TypeKindToString(expectedType.Kind)+", got bool")
			}

			operandType := inferComparisonOperandType(scope, e.Left, e.Right)

			if !isIntegerType(operandType) && !isFloatType(operandType) {
				ctx.addError(e.NodeMeta, "operator "+ast.TokenTypeToString(e.Op)+" requires integer or float operands")
				return
			}

			// Comparison operators always result in a bool type.
			validateExprType(ctx, scope, operandType, e.Left)
			validateExprType(ctx, scope, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		default:
			panic(fmt.Sprintf("unexpected binary operator %s", e.Op))
		}
	case *ast.BoolLiteralExpr:
		if expectedType.Kind != ast.TypeBool {
			ctx.addError(e.NodeMeta, "type mismatch: expected "+ast.TypeKindToString(expectedType.Kind)+", got bool")
		}
	case *ast.IdentExpr:
		// Types must match the declared type of the variable.
		declType, ok := scope[e.Name]
		if !ok {
			ctx.addError(e.NodeMeta, "undefined variable: "+e.Name)
			return
		}

		if declType.Kind != expectedType.Kind {
			ctx.addError(e.NodeMeta, "type mismatch: expected "+ast.TypeKindToString(expectedType.Kind)+", got "+ast.TypeKindToString(declType.Kind))
		}
	default:
		panic(fmt.Sprintf("unexpected expression type %T in validateExprType", expr))
	}
}

type intBounds struct {
	max    uint64
	signed bool
	name   string
}

// boundsForType returns the integer bounds for a given type, if it's an integer type.
func boundsForType(t *ast.TypeRef) (intBounds, bool) {
	switch t.Kind {
	case ast.TypeInt, ast.TypeInt64:
		return intBounds{max: uint64(math.MaxInt64), signed: true, name: "int"}, true
	case ast.TypeInt32:
		return intBounds{max: uint64(math.MaxInt32), signed: true, name: "int32"}, true
	case ast.TypeInt16:
		return intBounds{max: uint64(math.MaxInt16), signed: true, name: "int16"}, true
	case ast.TypeInt8:
		return intBounds{max: uint64(math.MaxInt8), signed: true, name: "int8"}, true
	case ast.TypeUInt64:
		return intBounds{max: math.MaxUint64, signed: false, name: "uint64"}, true
	case ast.TypeUInt32:
		return intBounds{max: math.MaxUint32, signed: false, name: "uint32"}, true
	case ast.TypeUInt16:
		return intBounds{max: math.MaxUint16, signed: false, name: "uint16"}, true
	case ast.TypeUInt8:
		return intBounds{max: math.MaxUint8, signed: false, name: "uint8"}, true
	default:
		return intBounds{}, false
	}
}

func literalFitsType(lit *ast.IntLiteralExpr, negative bool, target *ast.TypeRef) error {
	b, ok := boundsForType(target)
	if !ok {
		return errors.New("type is not an integer type")
	}

	u, err := strconv.ParseUint(lit.Literal, lit.Base, 64)
	if err != nil {
		return fmt.Errorf("invalid integer literal %q", lit.Literal)
	}

	// signed checks
	if b.signed {
		if negative {
			if u > b.max+1 {
				return fmt.Errorf("integer literal -%s out of range for %s", lit.Literal, b.name)
			}
			return nil
		}
		if u > b.max {
			return fmt.Errorf("integer literal %s out of range for %s", lit.Literal, b.name)
		}
		return nil
	}

	// unsigned checks
	if negative {
		return fmt.Errorf("cannot assign negative integer literal -%s to unsigned type %s", lit.Literal, b.name)
	}
	if u > b.max {
		return fmt.Errorf("integer literal %s out of range for %s", lit.Literal, b.name)
	}
	return nil
}

// floatLiteralFitsType checks if a float literal can fit in the expected type.
func floatLiteralFitsType(lit *ast.FloatLiteralExpr, target *ast.TypeRef) error {
	switch target.Kind {
	case ast.TypeFloat32:
		if _, err := strconv.ParseFloat(lit.Literal, 32); err != nil {
			return fmt.Errorf("invalid float literal %q for type float32", lit.Literal)
		}

	case ast.TypeFloat64, ast.TypeFloat:
		if _, err := strconv.ParseFloat(lit.Literal, 64); err != nil {
			return fmt.Errorf("invalid float literal %q for type %s", lit.Literal, ast.TypeToString(target))
		}

	default:
		return fmt.Errorf("type %s is not a float type", ast.TypeToString(target))
	}

	return nil
}

// inferComparisonOperandType checks the left and right expressions of a comparison operator to see if either has a known type, and returns that type if so. If neither has a known type, it defaults to int.
func inferComparisonOperandType(scope map[string]*ast.TypeRef, left, right ast.Expr) *ast.TypeRef {
	if t := exprKnownType(scope, left); t != nil {
		return t
	}
	if t := exprKnownType(scope, right); t != nil {
		return t
	}

	// Check for floats.
	if _, ok := left.(*ast.FloatLiteralExpr); ok {
		return ast.TypeFloatRef
	}
	if _, ok := right.(*ast.FloatLiteralExpr); ok {
		return ast.TypeFloatRef
	}

	// Must be ints if neither side is a float and we don't have any other information.
	return ast.TypeIntRef
}

// exprKnownType checks if the expression is a literal or identifier with a known type, and returns that type if so.
func exprKnownType(scope map[string]*ast.TypeRef, expr ast.Expr) *ast.TypeRef {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return scope[e.Name]
	case *ast.IntLiteralExpr:
		return e.InferredType
	case *ast.FloatLiteralExpr:
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

func isIntegerType(t *ast.TypeRef) bool {
	if t == nil {
		return false
	}

	switch t.Kind {
	case ast.TypeInt, ast.TypeInt64, ast.TypeInt32, ast.TypeInt16, ast.TypeInt8,
		ast.TypeUInt64, ast.TypeUInt32, ast.TypeUInt16, ast.TypeUInt8,
		ast.TypeByte, ast.TypeRune:
		return true
	default:
		return false
	}
}

func isFloatType(t *ast.TypeRef) bool {
	if t == nil {
		return false
	}

	switch t.Kind {
	case ast.TypeFloat, ast.TypeFloat32, ast.TypeFloat64:
		return true
	default:
		return false
	}
}

// inferExprDefaultType is used to infer the default type of an expression when we don't have any other information about what type it should be.
// This is used for literals and binary expressions where the type can be inferred from the context.
func inferExprDefaultType(ctx *validateContext, scope map[string]*ast.TypeRef, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.StringLiteralExpr:
		validateExprType(ctx, scope, ast.TypeStringRef, e)
	case *ast.BoolLiteralExpr:
		validateExprType(ctx, scope, ast.TypeBoolRef, e)
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EqEq, token.NotEq, token.Lt, token.LtEq, token.Gt, token.GtEq:
			validateExprType(ctx, scope, ast.TypeBoolRef, e)
		case token.Add, token.Sub, token.Mul, token.Div:
			if exprLooksFloat(scope, e.Left) || exprLooksFloat(scope, e.Right) {
				validateExprType(ctx, scope, ast.TypeFloatRef, e)
			} else {
				validateExprType(ctx, scope, ast.TypeIntRef, e)
			}
		default:
			validateExprType(ctx, scope, ast.TypeIntRef, e)
		}
	case *ast.FloatLiteralExpr:
		validateExprType(ctx, scope, ast.TypeFloatRef, e)
	default:
		validateExprType(ctx, scope, ast.TypeIntRef, expr)
	}
}

// exprLooksFloat checks if the expression is a float literal or an identifier with a known float type in the current scope.
func exprLooksFloat(scope map[string]*ast.TypeRef, expr ast.Expr) bool {
	if t := exprKnownType(scope, expr); t != nil {
		return isFloatType(t)
	}

	_, ok := expr.(*ast.FloatLiteralExpr)
	return ok
}
