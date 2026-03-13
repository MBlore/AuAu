package ir

import (
	"fmt"
	"math"
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

	// scopes tracks block-local variable bindings from innermost to outermost.
	scopes []map[string]varInfo

	// loops is a stack used for entering and exiting loops,
	// to help break/continue statements find the correct target blocks.
	loops []loopContext
}

// CompileFile converts the AST to IR. This is a simple traversal that emits IR instructions based on the AST nodes.
func CompileFile(file *ast.File) (*IRProgram, error) {
	prog := &IRProgram{}

	for _, ext := range file.Externs {
		irExt, err := buildExtern(ext)
		if err != nil {
			return nil, err
		}

		prog.Externs = append(prog.Externs, irExt)
	}

	for _, fn := range file.Functions {
		irFn, err := buildFunction(fn)
		if err != nil {
			return nil, err
		}

		prog.Functions = append(prog.Functions, irFn)
	}

	return prog, nil
}

func buildExtern(ext *ast.ExternFuncStmt) (*Extern, error) {
	irExt := &Extern{
		Name:   ext.Name,
		Args:   []Type{},
		Return: Type{},
	}

	for _, argType := range ext.Params {
		irExt.Args = append(irExt.Args, irTypeFromAstType(argType.Type))
	}

	irExt.Return = irTypeFromAstType(ext.ReturnType)

	return irExt, nil
}

// buildFunction creates a new IR function and emits instructions for the function body.
func buildFunction(fn *ast.FuncDecl) (*Function, error) {
	// Builder helps emit opcodes and the lowerer holds state across the process.
	builder := NewBuilder(fn.Name, fn.IsPublic)

	l := &Lowerer{
		builder: builder,
	}

	// We have to check if the main outer block of the function has a return.
	// Its valid for a void return function to not have one in the AST but we have
	// ensure at the IR/Backend level that all functions have a return instruction.
	hasReturn := false

	for _, st := range fn.Body.Stmts {
		if _, ok := st.(*ast.ReturnStmt); ok {
			hasReturn = true
			break
		}
	}

	// Create function-scope variables for the parameters before the body so they can be used in the body.
	l.pushScope()
	defer l.popScope()

	for i, param := range fn.Params {
		paramType := irTypeFromAstType(param.Type)
		paramVal := l.builder.Param(paramType, i)
		addr := l.builder.Alloc(paramType)

		l.currentScope()[param.Name] = varInfo{addr: addr, typ: paramType}

		// Store the parameter value in its address so it can be loaded later.
		l.builder.Store(addr, paramVal)
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
	l.pushScope()
	defer l.popScope()

	for _, stmt := range block.Stmts {
		if err := l.emitStmt(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (l *Lowerer) emitStmt(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.BreakStmt:
		loopCtx, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("break statement not inside a loop")
		}

		l.builder.Jump(loopCtx.breakBlock)

	case *ast.ContinueStmt:
		loopCtx, ok := l.currentLoop()
		if !ok {
			return fmt.Errorf("continue statement not inside a loop")
		}

		l.builder.Jump(loopCtx.continueBlock)

	case *ast.AssignStmt:
		// Emit instructions for the right-hand side expression.
		val, err := l.emitExpr(s.Value)
		if err != nil {
			return fmt.Errorf("invalid expression in assignment: %w", err)
		}

		// Look up the variable's address.
		v, ok := l.lookupVar(s.Name)
		if !ok {
			return fmt.Errorf("undefined variable: %s", s.Name)
		}

		// Emit a store instruction to update the variable's value.
		l.builder.Store(v.addr, val)

	case *ast.ForStmt:
		return l.emitForLoop(s)

	case *ast.WhileStmt:
		condBlock := l.builder.NewBlock("while_cond")
		bodyBlock := l.builder.NewBlock("while_body")
		endBlock := l.builder.NewBlock("while_end")

		l.pushLoop(endBlock, condBlock)
		defer l.popLoop()

		l.builder.Jump(condBlock)
		l.builder.SetBlock(condBlock)

		condVal, err := l.emitExpr(s.Cond)
		if err != nil {
			return fmt.Errorf("invalid condition expression in while statement: %w", err)
		}

		l.builder.Branch(condVal, bodyBlock, endBlock)
		l.builder.SetBlock(bodyBlock)

		if err := l.emitBlock(s.Body); err != nil {
			return err
		}

		// If the body doesn't end with a return or branch, add a jump back to the condition.
		if !blockTerminated(l.builder.CurrentBlock()) {
			l.builder.Jump(condBlock)
		}

		l.builder.SetBlock(endBlock)

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
		if _, exists := l.currentScope()[s.Name]; exists {
			// This should have been caught by the semantic phase.
			panic(fmt.Sprintf("variable %s already declared", s.Name))
		}

		l.currentScope()[s.Name] = varInfo{addr: addr, typ: irTypeFromAstType(s.Type)}

		// If there is an initializer, emit instructions to compute its value and store it.
		if s.Init != nil {
			val, err := l.emitExpr(s.Init)
			if err != nil {
				return err
			}

			// Emit a store instruction to initialize the variable.
			l.builder.Store(addr, val)
		}
	case *ast.IfStmt:

		// If without else.
		if s.Else == nil {
			thenBlock := l.builder.NewBlock("then")
			endBlock := l.builder.NewBlock("end")

			condVal, err := l.emitExpr(s.Cond)
			if err != nil {
				return fmt.Errorf("invalid condition expression in if statement: %w", err)
			}

			l.builder.Branch(condVal, thenBlock, endBlock)

			l.builder.SetBlock(thenBlock)
			if err := l.emitBlock(s.Then); err != nil {
				return err
			}

			// If the then block doesn't end with a return or branch, add a jump to the end.
			if !blockTerminated(l.builder.CurrentBlock()) {
				l.builder.Jump(endBlock)
			}

			l.builder.SetBlock(endBlock)
		} else {
			// If with else.
			thenBlock := l.builder.NewBlock("then")
			elseBlock := l.builder.NewBlock("else")
			endBlock := l.builder.NewBlock("end")

			condVal, err := l.emitExpr(s.Cond)
			if err != nil {
				return fmt.Errorf("invalid condition expression in if statement: %w", err)
			}

			l.builder.Branch(condVal, thenBlock, elseBlock)

			l.builder.SetBlock(thenBlock)
			if err := l.emitBlock(s.Then); err != nil {
				return err
			}

			// If the then block doesn't end with a return or branch, add a jump to the end.
			if !blockTerminated(l.builder.CurrentBlock()) {
				l.builder.Jump(endBlock)
			}

			l.builder.SetBlock(elseBlock)
			if err := l.emitStmt(s.Else); err != nil {
				return err
			}

			// If the else block doesn't end with a return or branch, add a jump to the end.
			if !blockTerminated(l.builder.CurrentBlock()) {
				l.builder.Jump(endBlock)
			}

			l.builder.SetBlock(endBlock)
		}

	case *ast.BlockStmt:
		if err := l.emitBlock(s); err != nil {
			return err
		}

	default:
		panic(fmt.Sprintf("unsupported statement type %T", stmt))
	}

	return nil
}

func (l *Lowerer) emitExpr(expr ast.Expr) (IRValue, error) {
	switch e := expr.(type) {
	case *ast.FloatLiteralExpr:
		t := irTypeFromAstType(e.InferredType)

		// Float bits are stored as a uint64 const.
		switch t.Kind {
		case TypeFloat32:
			f, err := strconv.ParseFloat(e.Literal, 32)
			if err != nil {
				return 0, fmt.Errorf("invalid float literal %s: %w", e.Literal, err)
			}

			bits := math.Float32bits(float32(f))
			return l.builder.Const(t, uint64(bits)), nil

		case TypeFloat64:
			f, err := strconv.ParseFloat(e.Literal, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid float literal %s: %w", e.Literal, err)
			}

			bits := math.Float64bits(f)
			return l.builder.Const(t, bits), nil

		default:
			return 0, fmt.Errorf("unsupported float type %v", e.InferredType)
		}

	case *ast.BoolLiteralExpr:
		return l.builder.Const(Type{Kind: TypeBool}, boolToInt(e.Value)), nil
	case *ast.StringLiteralExpr:
		strVal := l.builder.StringConst([]byte(e.Value))
		return strVal, nil
	case *ast.IdentExpr:
		// Look up the variable name and emit a load from its address.
		v, ok := l.lookupVar(e.Name)
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

func (l *Lowerer) pushScope() {
	l.scopes = append(l.scopes, make(map[string]varInfo))
}

func (l *Lowerer) popScope() {
	if len(l.scopes) == 0 {
		panic("popScope called with empty scope stack")
	}

	l.scopes = l.scopes[:len(l.scopes)-1]
}

func (l *Lowerer) currentScope() map[string]varInfo {
	if len(l.scopes) == 0 {
		panic("currentScope called with empty scope stack")
	}

	return l.scopes[len(l.scopes)-1]
}

func (l *Lowerer) lookupVar(name string) (varInfo, bool) {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if v, ok := l.scopes[i][name]; ok {
			return v, true
		}
	}

	return varInfo{}, false
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
	case ast.TypeByte:
		return Type{Kind: TypeU8}
	case ast.TypeRune:
		return Type{Kind: TypeI32}
	case ast.TypeVoid:
		return Type{Kind: TypeVoid}
	case ast.TypeFloat32:
		return Type{Kind: TypeFloat32}
	case ast.TypeFloat64:
		return Type{Kind: TypeFloat64}
	case ast.TypeFloat:
		return Type{Kind: TypeFloat64}
	default:
		panic(fmt.Sprintf("unsupported AST type %d", astType.Kind))
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

// blockTerminated returns true if the last instruction is OpReturn, OpBranch, or OpJump.
func blockTerminated(block *Block) bool {
	if len(block.Instrs) == 0 {
		return false
	}

	lastOp := block.Instrs[len(block.Instrs)-1].Op
	return lastOp == OpReturn || lastOp == OpBranch || lastOp == OpJump
}
