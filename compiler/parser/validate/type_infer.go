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
		if e.Op != token.Minus {
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
		// TODO: Validate operators are valid for the types.
		// For binary expressions, we need to validate both sides.
		validateExprType(ctx, expectedType, e.Left)
		validateExprType(ctx, expectedType, e.Right)
		e.InferredType = expectedType
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
