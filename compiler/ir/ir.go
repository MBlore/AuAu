package ir

import (
	"fmt"
	"math"
	"strconv"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

// varInfo holds the address and type of a variable in the current scope.
type varInfo struct {
	addr IRValue
	typ  Type
}

// abiParam represents a parameter in the function signature, used for externs and function definitions.
type abiParam struct {
	name string
	typ  Type
}

// fnSig represents the signature of a function, including its argument types, return type, and whether it's an extern.
type fnSig struct {
	args   []Type
	ret    Type
	extern bool
}

// Lowerer holds state for a single build file pass.
type Lowerer struct {
	builder *Builder

	// scopes tracks block-local variable bindings from innermost to outermost.
	scopes []map[string]varInfo

	// loops is a stack used for entering and exiting loops,
	// to help break/continue statements find the correct target blocks.
	loops []loopContext

	// funcs tracks function signatures for all defined functions.
	funcs map[string]fnSig

	// For functions that return aggregate-like types (currently just strings)
	// we need to pass a hidden sret pointer parameter for the return value.
	// This tracks the IRValue for that hidden parameter if it exists so we can
	// store the return value to it before returning.
	returnAddr IRValue

	// hasReturnAddr tracks whether the current function has a hidden sret pointer parameter.
	// This is needed to know whether to store the return value to the returnAddr before returning.
	hasReturnAddr bool
}

// CompileFile converts the AST to IR. This is a simple traversal that emits IR instructions based on the AST nodes.
func CompileFile(file *ast.File) (*IRProgram, error) {
	prog := &IRProgram{}

	funcs := make(map[string]fnSig)

	for _, ext := range file.Externs {
		irExt, err := buildExtern(ext)
		if err != nil {
			return nil, err
		}

		// Record the function sig.
		funcs[irExt.Name] = fnSig{
			args:   irExt.Args,
			ret:    irExt.Return,
			extern: true,
		}

		prog.Externs = append(prog.Externs, irExt)
	}

	// Record function signatures for all functions before building them, so that calls to other functions can find the sigs.
	for _, fn := range file.Functions {
		args := make([]Type, 0, len(fn.Params))
		for _, p := range fn.Params {
			args = append(args, irTypeFromAstType(p.Type))
		}

		funcs[fn.Name] = fnSig{
			args: args,
			ret:  irTypeFromAstType(fn.ReturnType),
		}
	}

	for _, fn := range file.Functions {
		irFn, err := buildFunction(fn, funcs)
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
func buildFunction(fn *ast.FuncDecl, funcs map[string]fnSig) (*Function, error) {
	// Builder helps emit opcodes and the lowerer holds state across the process.
	builder := NewBuilder(fn.Name, fn.IsPublic)

	l := &Lowerer{
		builder: builder,
		funcs:   funcs,
	}

	// Create function-scope variables for the parameters before the body so they can be used in the body.
	l.pushScope()
	defer l.popScope()

	// Emit param binds.
	if err := l.bindFunctionParams(fn); err != nil {
		return nil, err
	}

	// Emit the body.
	if err := l.emitBlock(fn.Body); err != nil {
		return nil, err
	}

	retType := irTypeFromAstType(fn.ReturnType)

	// Only void functions get an implicit trailing return.
	if retType.Kind == TypeVoid && !blockTerminated(builder.CurrentBlock()) {
		builder.Return()
	}

	return builder.Function(), nil
}

// buildStackFrame constructs a stack frame for the given function by looking for all instructions
// that produce values that need to be stored on the stack. It tracks the offset and type of
// each value for use in code generation.
func (l *Lowerer) bindFunctionParams(fn *ast.FuncDecl) error {
	abiParams := flattenFunctionABIParams(fn)
	abiIndex := 0
	retType := irTypeFromAstType(fn.ReturnType)

	// Use sret pointer for struct returns.
	if ReturnsViaHiddenPtr(retType) {
		retPtrVal := l.builder.Param(abiParams[abiIndex].typ, abiIndex)
		l.returnAddr = retPtrVal
		l.hasReturnAddr = true
		abiIndex++
	}

	for _, param := range fn.Params {
		paramType := irTypeFromAstType(param.Type)
		addr := l.builder.Alloc(paramType)

		// Check and remember the parameter in the current scope.
		if _, exists := l.currentScope()[param.Name]; exists {
			panic(fmt.Sprintf("parameter %s already declared", param.Name))
		}

		l.currentScope()[param.Name] = varInfo{
			addr: addr,
			typ:  paramType,
		}

		// Single parameter case, just bind it directly.
		if len(flattenABIType(paramType)) == 1 {
			paramVal := l.builder.Param(abiParams[abiIndex].typ, abiIndex)
			abiIndex++
			l.builder.Store(addr, paramVal)
			continue
		}

		// For flattened types, we need to bind each field separately.
		l.bindAggregateParam(addr, paramType, abiParams, &abiIndex)
	}

	return nil
}

// bindAggregateParam binds a struct parameter by recursively binding each field.
func (l *Lowerer) bindAggregateParam(addr IRValue, t Type, abiParams []abiParam, abiIndex *int) {
	if !t.IsStruct() {
		// Single parameter case, just bind it directly.
		paramVal := l.builder.Param(abiParams[*abiIndex].typ, *abiIndex)

		*abiIndex = *abiIndex + 1

		l.builder.Store(addr, paramVal)
		return
	}

	// Multiple parameter case, we need to bind each field separately
	// and store them to the correct offset in the struct.
	for i, field := range t.Fields {
		fieldAddr := l.builder.FieldAddr(addr, fieldOffset(t, i), field.Type)
		l.bindAggregateParam(fieldAddr, field.Type, abiParams, abiIndex)
	}
}

// buildStackFrame constructs a stack frame for the given function by looking for all instructions
// that produce values that need to be stored on the stack. It tracks the offset and type of
// each value for use in code generation.
func flattenFunctionABIParams(fn *ast.FuncDecl) []abiParam {
	params := make([]abiParam, 0, len(fn.Params)+1)

	retType := irTypeFromAstType(fn.ReturnType)

	// Hidden sret pointer for struct returns.
	if ReturnsViaHiddenPtr(retType) {
		params = append(params, abiParam{
			name: "__ret_ptr",
			typ:  PtrType(retType),
		})
	}

	for _, param := range fn.Params {
		t := irTypeFromAstType(param.Type)
		flat := flattenABIType(t)

		// If the type doesn't flatten to multiple types, we can just pass it as a single parameter.
		if len(flat) == 1 {
			params = append(params, abiParam{
				name: param.Name,
				typ:  flat[0],
			})

			continue
		}

		// For flattened types, we need to pass each flattened field as a separate parameter.
		for i, fieldType := range flat {
			params = append(params, abiParam{
				name: fmt.Sprintf("%s.__%d", param.Name, i),
				typ:  fieldType,
			})
		}
	}

	return params
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
			// Look up the function signature for the called function.
			sig, ok := l.funcs[s.FuncName]
			if !ok {
				return fmt.Errorf("unknown function: %s", s.FuncName)
			}

			// Check that the number of arguments matches the function signature.
			if len(sig.args) != len(s.Args) {
				return fmt.Errorf("function %s expects %d arguments, got %d", s.FuncName, len(sig.args), len(s.Args))
			}

			// Emit instructions for each argument expression and collect their IRValues.
			args := make([]IRValue, 0, len(s.Args))
			for _, argExpr := range s.Args {
				argVal, err := l.emitExpr(argExpr)
				if err != nil {
					return fmt.Errorf("invalid argument to %s: %w", s.FuncName, err)
				}
				args = append(args, argVal)
			}

			// Emit the call instruction.
			if sig.ret.Kind == TypeVoid {
				l.builder.CallVoid(s.FuncName, args...)
			} else {
				_ = l.builder.Call(s.FuncName, sig.ret, args...)
			}
		}
	case *ast.ReturnStmt:
		if s.ReturnExpr != nil {
			// Emit instructions for the return value expression.
			val, err := l.emitExpr(s.ReturnExpr)
			if err != nil {
				return err
			}

			// If the function has a hidden sret pointer parameter, store the return value to
			// it before returning.
			if l.hasReturnAddr {
				l.builder.Store(l.returnAddr, val)
				l.builder.Return()
			} else {
				l.builder.Return(val)
			}
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
	case *ast.CallExpr:
		// Emit instructions for the function call.
		sig, ok := l.funcs[e.FuncName]
		if !ok {
			return 0, fmt.Errorf("unknown function: %s", e.FuncName)
		}

		// Check that the number of arguments matches the function signature.
		if len(sig.args) != len(e.Args) {
			return 0, fmt.Errorf("function %s expects %d arguments, got %d", e.FuncName, len(sig.args), len(e.Args))
		}

		// Emit instructions for each argument expression and collect their IRValues.
		args := make([]IRValue, 0, len(e.Args))

		for _, argExpr := range e.Args {
			argVal, err := l.emitExpr(argExpr)

			if err != nil {
				return 0, fmt.Errorf("invalid argument to %s: %w", e.FuncName, err)
			}

			args = append(args, argVal)
		}

		// Dont allow function calls in expressions if the function returns void.
		if sig.ret.Kind == TypeVoid {
			return 0, fmt.Errorf("void function %s cannot be used as an expression", e.FuncName)
		}

		return l.builder.Call(e.FuncName, sig.ret, args...), nil

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
		return StringType()
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

// flattenABIType takes a Type and if it's a struct, it recursively flattens its fields into a slice of Types.
func flattenABIType(t Type) []Type {
	if !t.IsStruct() {
		return []Type{t}
	}

	var types []Type
	for _, field := range t.Fields {
		types = append(types, flattenABIType(field.Type)...)
	}

	return types
}

func abiSize(t Type) int {
	if !t.IsStruct() {
		// All non-struct types are 8 bytes in size in our ABI for simplicity.
		// Later we can optimize this using smaller sizes for certain types if we want.
		return 8
	}

	size := 0

	for _, field := range t.Fields {
		size += abiSize(field.Type)
	}

	return size
}

// fieldOffset calculates the byte offset of a field within a struct type, accounting for nested structs.
func fieldOffset(t Type, fieldIndex int) int {
	if !t.IsStruct() {
		panic("fieldOffset called on non-struct type")
	}

	offset := 0

	for i := 0; i < fieldIndex; i++ {
		offset += abiSize(t.Fields[i].Type)
	}

	return offset
}
