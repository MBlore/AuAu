package ir

func NewBuilder(name string) *Builder {
	fn := &Function{Name: name}

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

// Emit creates a new instruction with the given opcode and arguments, and returns the destination value.
func (b *Builder) Emit(op OpCode, args ...IRValue) IRValue {
	if b.current == nil {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: no current block")
	}

	// In true SSA form, each instruction produces a new value.
	dest := b.NewValue()

	instr := &Instr{
		Op:   op,
		Dest: dest,
		Args: args,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

// Const creates a new constant instruction and returns the destination value.
func (b *Builder) Const(value int64) IRValue {
	// Every constant also produces a new value (SSA form).
	dest := b.NewValue()

	instr := &Instr{
		Op:    OpConst,
		Dest:  dest,
		Const: value,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

// Store creates a new store instruction to store a value at a given address.
func (b *Builder) Store(addr IRValue, val IRValue) {
	if b.current == nil {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: no current block")
	}

	instr := &Instr{
		Op:   OpStore,
		Args: []IRValue{addr, val},
	}

	b.current.Instrs = append(b.current.Instrs, instr)
}

// Load creates a new load instruction to load a value from a given address, and returns the loaded value.
func (b *Builder) Load(addr IRValue) IRValue {
	if b.current == nil {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: no current block")
	}

	dest := b.NewValue()
	instr := &Instr{
		Op:   OpLoad,
		Dest: dest,
		Args: []IRValue{addr},
	}

	b.current.Instrs = append(b.current.Instrs, instr)
	return dest
}

// Ret creates a new return instruction with the given values.
func (b *Builder) Ret(vals ...IRValue) {
	if len(vals) > 1 {
		// Panic here, as it should have been caught by the semantic analysis phase.
		panic("IR builder: multiple return values not supported yet")
	}

	instr := &Instr{
		Op:   OpRet,
		Args: vals,
	}

	b.current.Instrs = append(b.current.Instrs, instr)
}

// Alloc creates a new allocation instruction and returns the address of the allocated memory.
func (b *Builder) Alloc() IRValue {
	return b.Emit(OpAlloc)
}
