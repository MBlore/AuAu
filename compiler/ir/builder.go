package ir

func NewBuilder(name string, public bool) *Builder {
	fn := &Function{Name: name, Public: public}

	b := &Builder{
		fn: fn,
	}

	entry := b.NewBlock()
	b.SetBlock(entry)
	return b
}

func (b *Builder) Function() *Function {
	return b.fn
}

func (b *Builder) SetBlock(block *Block) {
	b.current = block
}

func (b *Builder) NewBlock() *Block {
	block := &Block{ID: b.nextBlockID}
	b.nextBlockID++
	b.fn.Blocks = append(b.fn.Blocks, block)
	return block
}

func (b *Builder) NewValue() IRValue {
	val := b.nextVal
	b.nextVal++
	return val
}

func (b *Builder) ensureBlock() {
	if b.current == nil {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: no current block")
	}
}

// emitValue creates a new instruction with the given opcode and arguments, and returns the destination value.
func (b *Builder) emitValue(op OpCode, tp Type, args ...IRValue) IRValue {
	b.ensureBlock()

	// In true SSA form, each instruction produces a new value.
	dest := b.NewValue()

	instr := &Instr{
		Op:   op,
		Dest: dest,
		Type: tp,
		Args: args,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

func (b *Builder) Add(tp Type, l, r IRValue) IRValue {
	return b.emitValue(OpAdd, tp, l, r)
}

func (b *Builder) Sub(tp Type, l, r IRValue) IRValue {
	return b.emitValue(OpSub, tp, l, r)
}

func (b *Builder) Mul(tp Type, l, r IRValue) IRValue {
	return b.emitValue(OpMul, tp, l, r)
}

func (b *Builder) Div(tp Type, l, r IRValue) IRValue {
	return b.emitValue(OpDiv, tp, l, r)
}

func (b *Builder) Neg(tp Type, val IRValue) IRValue {
	return b.emitValue(OpNeg, tp, val)
}

// Const creates a new constant instruction and returns the destination value.
func (b *Builder) Const(tp Type, value int64) IRValue {
	b.ensureBlock()

	// Every constant also produces a new value (SSA form).
	dest := b.NewValue()

	instr := &Instr{
		Op:    OpConst,
		Dest:  dest,
		Type:  tp,
		Const: value,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

// Store creates a new store instruction to store a value at a given address.
func (b *Builder) Store(addr IRValue, val IRValue) {
	b.ensureBlock()

	instr := &Instr{
		Op:   OpStore,
		Args: []IRValue{addr, val},
	}

	b.current.Instrs = append(b.current.Instrs, instr)
}

// Load creates a new load instruction to load a value from a given address, and returns the loaded value.
func (b *Builder) Load(tp Type, addr IRValue) IRValue {
	b.ensureBlock()

	dest := b.NewValue()

	instr := &Instr{
		Op:   OpLoad,
		Dest: dest,
		Type: tp,
		Args: []IRValue{addr},
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

// Return creates a new return instruction with the given values.
func (b *Builder) Return(vals ...IRValue) {
	b.ensureBlock()

	if len(vals) > 1 {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: multiple return values not supported yet")
	}

	instr := &Instr{
		Op:   OpReturn,
		Args: vals,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
}

// PtrType creates a pointer type for the given element type.
func PtrType(elem Type) Type {
	return Type{
		Kind: TypePtr,
		Elem: &elem,
	}
}

// Alloc creates a new allocation instruction and returns the address of the allocated memory.
func (b *Builder) Alloc(elem Type) IRValue {
	return b.emitValue(OpAlloc, PtrType(elem))
}
