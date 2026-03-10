package ir

import (
	"fmt"
	"strconv"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

type varInfo struct {
	addr IRValue
	typ  Type
}

// Lowerer holds state for a single build file pass.
type Lowerer struct {
	builder *Builder
	// vars maps variable names to their corresponding IR values.
	vars map[string]varInfo
}

// CompileFile converts the AST to IR. This is a simple traversal that emits IR instructions based on the AST nodes.
func CompileFile(file *ast.File) (*IRProgram, error) {
	prog := &IRProgram{}

	for _, fn := range file.Functions {
		irFn, err := buildFunction(fn)
		if err != nil {
			return nil, err
		}

		prog.Functions = append(prog.Functions, irFn)
	}

	return prog, nil
}

// buildFunction creates a new IR function and emits instructions for the function body.
func buildFunction(fn *ast.FuncDecl) (*Function, error) {
	// Builder helps emit opcodes and the lowerer holds state across the process.
	builder := NewBuilder(fn.Name, fn.IsPublic)

	l := &Lowerer{
		builder: builder,
		vars:    make(map[string]varInfo),
	}

	// We have to check if the main outer block of the function has a return.
	// Its valid for a void return function to not have one in the AST but we have
	// ensure at the IR/Backend level that all functions have a return instruction.
	hasReturn := false

	// TODO: Returns are also valid in if/else blocks, loops, and possibly others.
	// Semantic analysis should ensure that all code paths in a non-void function
	// have a return, and that void functions don't return values.
	for _, st := range fn.Body.Stmts {
		if _, ok := st.(*ast.ReturnStmt); ok {
			hasReturn = true
			break
		}
	}

	// Now emit the function body block.
	err := l.emitBlock(fn.Body)
	if err != nil {
		return nil, err
	}

	// If there was no return, emit a default one.
	if !hasReturn {
		builder.Return()
	}

	return builder.Function(), nil
}

// emitBlock iterates over the statements in the given block and emits instructions.
func (l *Lowerer) emitBlock(block *ast.BlockStmt) error {
	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *ast.CallStmt:
			// Emit instructions for the function call.
			if s.FuncName == "print" {
				// Special case for print since we don't have a standard library yet.
				// We will emit a call to an OpPrint instruction that the backend can handle.
				if len(s.Args) != 1 {
					return fmt.Errorf("print expects exactly one argument")
				}

				val, err := l.emitExpr(s.Args[0])
				if err != nil {
					return fmt.Errorf("invalid argument to print: %w", err)
				}

				l.builder.Print(val)
			} else {
				panic(fmt.Sprintf("unsupported function call: %s", s.FuncName))
			}
		case *ast.ReturnStmt:
			if s.ReturnExpr != nil {
				val, err := l.emitExpr(s.ReturnExpr)
				if err != nil {
					return err
				}

				l.builder.Return(val)
			} else {
				l.builder.Return()
			}
		case *ast.VarDeclStmt:
			// Create a new address value for the variable.
			addr := l.builder.Alloc(irTypeFromAstType(s.Type))

			// Remember the variable name and its address.
			if _, exists := l.vars[s.Name]; exists {
				// This should have been caught by the semantic phase.
				panic(fmt.Sprintf("variable %s already declared", s.Name))
			}

			l.vars[s.Name] = varInfo{addr: addr, typ: irTypeFromAstType(s.Type)}

			// If there is an initializer, emit instructions to compute its value and store it.
			if s.Init != nil {
				val, err := l.emitExpr(s.Init)
				if err != nil {
					return err
				}

				// Emit a store instruction to initialize the variable.
				l.builder.Store(addr, val)
			}
		default:
			panic(fmt.Sprintf("unsupported statement type %T", stmt))
		}
	}

	return nil
}

func (l *Lowerer) emitExpr(expr ast.Expr) (IRValue, error) {
	switch e := expr.(type) {
	case *ast.BoolLiteralExpr:
		return l.builder.Const(Type{Kind: TypeBool}, boolToInt(e.Value)), nil
	case *ast.StringLiteralExpr:
		strVal := l.builder.StringConst([]byte(e.Value))
		return strVal, nil
	case *ast.IdentExpr:
		// Look up the variable name and emit a load from its address.
		v, ok := l.vars[e.Name]
		if !ok {
			return 0, fmt.Errorf("undefined variable: %s", e.Name)
		}

		return l.builder.Load(v.typ, v.addr), nil
	case *ast.IntLiteralExpr:
		val, err := strconv.ParseUint(e.Literal, e.Base, 64)

		if err != nil {
			return 0, fmt.Errorf("invalid integer literal %s: %w", e.Literal, err)
		}

		t := irTypeFromAstType(e.InferredType)

		return l.builder.Const(t, val), nil
	case *ast.UnaryExpr:
		// This handles cases where a unary op appears in any expression.
		val, err := l.emitExpr(e.Expr)
		if err != nil {
			return 0, err
		}

		switch e.Op {
		case token.Sub:
			t := irTypeFromAstType(e.InferredType)
			return l.builder.Neg(t, val), nil
		default:
			return 0, fmt.Errorf("unsupported unary operator %s", e.Op)
		}
	case *ast.BinaryExpr:
		// This handles cases such as 'int a = <expr>'.
		left, err := l.emitExpr(e.Left)
		if err != nil {
			return 0, err
		}

		right, err := l.emitExpr(e.Right)
		if err != nil {
			return 0, err
		}

		switch e.Op {
		case token.Add, token.Sub, token.Mul, token.Div:
			// Emit the binary operation instruction.
			t := irTypeFromAstType(e.InferredType)
			switch e.Op {
			case token.Add:
				return l.builder.Add(t, left, right), nil
			case token.Sub:
				return l.builder.Sub(t, left, right), nil
			case token.Mul:
				return l.builder.Mul(t, left, right), nil
			case token.Div:
				return l.builder.Div(t, left, right), nil
			}
		case token.EqEq, token.NotEq, token.Lt, token.LtEq, token.Gt, token.GtEq:
			// Emit a comparison instruction.
			cmpKind, err := cmpKindFromToken(e.Op)
			if err != nil {
				return 0, err
			}

			return l.builder.Cmp(cmpKind, left, right), nil
		}

		return 0, fmt.Errorf("unsupported binary operator %s", ast.TokenTypeToString(e.Op))
	default:
		return 0, fmt.Errorf("unsupported expression type %T", e)
	}
}

func boolToInt(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// irTypeFromAstType converts an AST type to an IR type.
func irTypeFromAstType(astType *ast.TypeRef) Type {
	switch t := astType.Kind; t {
	case ast.TypeInt:
		return Type{Kind: TypeI64}
	case ast.TypeInt8:
		return Type{Kind: TypeI8}
	case ast.TypeInt16:
		return Type{Kind: TypeI16}
	case ast.TypeInt32:
		return Type{Kind: TypeI32}
	case ast.TypeInt64:
		return Type{Kind: TypeI64}
	case ast.TypeUInt8:
		return Type{Kind: TypeU8}
	case ast.TypeUInt16:
		return Type{Kind: TypeU16}
	case ast.TypeUInt32:
		return Type{Kind: TypeU32}
	case ast.TypeUInt64:
		return Type{Kind: TypeU64}
	case ast.TypeString:
		return Type{Kind: TypeString}
	case ast.TypeBool:
		return Type{Kind: TypeBool}
	default:
		panic(fmt.Sprintf("unsupported AST type %T", astType))
	}
}

// cmpKindFromToken converts an AST comparison operator token to a CmpKind.
func cmpKindFromToken(tok token.TokenType) (CmpKind, error) {
	switch tok {
	case token.EqEq:
		return CmpEq, nil
	case token.NotEq:
		return CmpNotEq, nil
	case token.Lt:
		return CmpLt, nil
	case token.LtEq:
		return CmpLtEq, nil
	case token.Gt:
		return CmpGt, nil
	case token.GtEq:
		return CmpGtEq, nil
	default:
		return 0, fmt.Errorf("unsupported comparison operator %s", ast.TokenTypeToString(tok))
	}
}
