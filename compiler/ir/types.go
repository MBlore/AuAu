package ir

type TypeKind int

const (
	TypeI32 TypeKind = iota
	TypeI64
	TypeI16
	TypeI8
	TypeU8
	TypeU16
	TypeU32
	TypeU64
	TypePtr
	TypeString
	TypeBool
	TypeVoid
	TypeFloat32
	TypeFloat64
)

type Type struct {
	Kind TypeKind
	Elem *Type
}

type CmpKind int

const (
	CmpEq CmpKind = iota
	CmpNotEq
	CmpLt
	CmpLtEq
	CmpGt
	CmpGtEq
)

type IRProgram struct {
	Functions []*Function
	Externs   []*Extern
}

// IRValue represents a value in the IR, which can be an SSA value or an address.
// This later can be somewhere in memory or a register.
// We remain agnostic about the storage location because we don't commit
// to any specific CPU architecture at IR stage. It is up to backend implementations
// to decide how to map IR values to actual registers or memory locations.
type IRValue int

type OpCode int

const (
	OpConst OpCode = iota
	OpStringConst
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpNeg
	OpLoad
	OpStore

	// OpCall requires the following on Instr:
	// - Callee string: the name of the function being called.
	// - Args []IRValue: the arguments to the function.
	// - Type: the return type of the function (for non-void).
	// - Dest: the destination value for the return value (for non-void).
	OpCall

	OpCmp
	OpBranch
	OpJump
	OpReturn
	OpAlloc

	// OpParam represents machine/ABI incoming params, not source-language params.
	OpParam

	OpPrint // temporary opcode for testing purposes, to be removed later when print can be extern imported from C.
)

type Instr struct {
	Op         OpCode    // The operation code of this instruction.
	Dest       IRValue   // The destination value of this instruction, if any.
	Type       Type      // The type of the value produced by this instruction, if any.
	Args       []IRValue // For OpLoad and OpStore, the address is stored here.
	Const      uint64    // For OpLoad and OpStore, the address is stored here.
	Data       []byte    // For OpStringConst, the string data is stored here.
	Cmp        CmpKind   // For OpCmp, the kind of comparison.
	TrueBlock  *Block    // For OpBranch, the block to jump to if the condition is true.
	FalseBlock *Block    // For OpBranch, the block to jump to if the condition is false.
	JumpBlock  *Block    // For OpJump, the block to jump to unconditionally.
	Callee     string    // For OpCall, the function being called.
	ParamIndex int       // For OpParam, the index of the incoming parameter.
}

type Function struct {
	Name   string
	Public bool
	Blocks []*Block
}

type Extern struct {
	Name   string
	Args   []Type
	Return Type
}

type Block struct {
	ID     int
	Name   string
	Instrs []*Instr
}

type Builder struct {
	// The function being built.
	fn *Function
	// The current active block.
	current *Block
	// SSA value counter.
	nextVal IRValue
	// Block numbering counter.
	nextBlockID int
}
