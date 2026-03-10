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
		inferTypesInBlock(ctx, fn.Body)
	}
}

func inferTypesInBlock(ctx *validateContext, block *ast.BlockStmt) {
	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *ast.VarDeclStmt:
			if s.Init != nil {
				validateExprType(ctx, s.Type, s.Init)
			}
		}
	}
}

// validateExprType ensures that the expression tree matches the expected type.
func validateExprType(ctx *validateContext, expectedType *ast.TypeRef, expr ast.Expr) {
	// We know from the declaration type what the expression should be,
	// but have to verify the literal will fit in that type and set the literal's type accordingly.

	// Now we look for IntLiteralExpr in the expression graph and set their type to the inferred type.
	switch e := expr.(type) {
	case *ast.StringLiteralExpr:
		if expectedType.Kind != ast.TypeString {
			ctx.errors = append(ctx.errors, fmt.Errorf("type mismatch: expected %s, got string",
				ast.TypeKindToString(expectedType.Kind)))
		}
	case *ast.IntLiteralExpr:
		if err := literalFitsType(e, false, expectedType); err != nil {
			ctx.errors = append(ctx.errors, err)
			return
		}

		e.InferredType = expectedType
	case *ast.UnaryExpr:
		if e.Op != token.Sub {
			return
		}

		lit, ok := e.Expr.(*ast.IntLiteralExpr)
		if !ok {
			// Its not an int literal, so skip.
			return
		}

		if err := literalFitsType(lit, true, expectedType); err != nil {
			ctx.errors = append(ctx.errors, err)
			return
		}

		lit.InferredType = expectedType
		e.InferredType = expectedType
	case *ast.BinaryExpr:
		switch e.Op {
		case token.EqEq, token.NotEq:
			if expectedType.Kind != ast.TypeBool {
				ctx.errors = append(ctx.errors, fmt.Errorf("type mismatch: expected %s, got bool",
					ast.TypeKindToString(expectedType.Kind)))
			}

			operandType := inferComparisonOperandType(ctx, e.Left, e.Right)

			validateExprType(ctx, operandType, e.Left)
			validateExprType(ctx, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		case token.Add, token.Sub, token.Mul, token.Div:
			// For binary expressions, we need to validate both sides.
			validateExprType(ctx, expectedType, e.Left)
			validateExprType(ctx, expectedType, e.Right)

			e.InferredType = expectedType
		case token.Lt, token.LtEq, token.Gt, token.GtEq:
			// Check expected type is valid for comparison operators.
			if expectedType.Kind != ast.TypeBool {
				ctx.errors = append(ctx.errors, fmt.Errorf("type mismatch: expected %s, got bool",
					ast.TypeKindToString(expectedType.Kind)))
			}

			operandType := inferComparisonOperandType(ctx, e.Left, e.Right)

			if !isIntegerType(operandType) {
				ctx.errors = append(ctx.errors, fmt.Errorf("operator %s requires integer operands", ast.TokenTypeToString(e.Op)))
				return
			}

			// Comparison operators always result in a bool type.
			validateExprType(ctx, operandType, e.Left)
			validateExprType(ctx, operandType, e.Right)

			e.InferredType = ast.TypeBoolRef
		default:
			panic(fmt.Sprintf("unexpected binary operator %s", e.Op))
		}
	case *ast.BoolLiteralExpr:
		if expectedType.Kind != ast.TypeBool {
			ctx.errors = append(ctx.errors, fmt.Errorf("type mismatch: expected %s, got bool",
				ast.TypeKindToString(expectedType.Kind)))
		}
	case *ast.IdentExpr:
		// Types must match the declared type of the variable.
		declType, ok := ctx.varTypes[e.Name]
		if !ok {
			ctx.errors = append(ctx.errors, fmt.Errorf("undefined variable: %s", e.Name))
			return
		}

		if declType.Kind != expectedType.Kind {
			ctx.errors = append(
				ctx.errors,
				fmt.Errorf("type mismatch: expected %s, got %s",
					ast.TypeKindToString(expectedType.Kind),
					ast.TypeKindToString(declType.Kind)))
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

// inferComparisonOperandType checks the left and right expressions of a comparison operator to see if either has a known type, and returns that type if so. If neither has a known type, it defaults to int.
func inferComparisonOperandType(ctx *validateContext, left, right ast.Expr) *ast.TypeRef {
	if t := exprKnownType(ctx, left); t != nil {
		return t
	}
	if t := exprKnownType(ctx, right); t != nil {
		return t
	}

	return ast.TypeIntRef
}

// exprKnownType checks if the expression is a literal or identifier with a known type, and returns that type if so.
func exprKnownType(ctx *validateContext, expr ast.Expr) *ast.TypeRef {
	switch e := expr.(type) {
	case *ast.IdentExpr:
		return ctx.varTypes[e.Name]
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
