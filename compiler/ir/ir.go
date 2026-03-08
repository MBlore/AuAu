package ir

import (
	"fmt"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

// Lowerer holds state for a single build file pass.
type Lowerer struct {
	builder *Builder
	// vars maps variable names to their corresponding IR values.
	vars map[string]IRValue
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
		vars:    make(map[string]IRValue),
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
			addr := l.builder.Alloc(Type{Kind: TypeI64})

			// Remember the variable name and its address.
			if _, exists := l.vars[s.Name]; exists {
				// This should have been caught by the semantic phase.
				panic(fmt.Sprintf("variable %s already declared", s.Name))
			}

			l.vars[s.Name] = addr

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
	case *ast.IntLiteralExpr:
		// This handles cases such as 'int a = 5'.
		return l.builder.Const(Type{Kind: TypeI64}, e.Value), nil
	case *ast.UnaryExpr:
		// This handles cases where a unary op appears in any expression.
		val, err := l.emitExpr(e.Expr)
		if err != nil {
			return 0, err
		}

		switch e.Op {
		case token.Minus:
			return l.builder.Neg(Type{Kind: TypeI64}, val), nil
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

		// Emit the binary operation instruction.
		switch e.Op {
		case token.Plus:
			return l.builder.Add(Type{Kind: TypeI64}, left, right), nil
		case token.Minus:
			return l.builder.Sub(Type{Kind: TypeI64}, left, right), nil
		case token.Asterisk:
			return l.builder.Mul(Type{Kind: TypeI64}, left, right), nil
		case token.Slash:
			return l.builder.Div(Type{Kind: TypeI64}, left, right), nil
		default:
			return 0, fmt.Errorf("unsupported binary operator %s", e.Op)
		}
	default:
		return 0, fmt.Errorf("unsupported expression type %T", e)
	}
}
