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
	OpCall
	OpCmp
	OpBranch
	OpJump
	OpReturn
	OpAlloc
	OpPrint // temporary opcode for testing purposes, to be removed later when print can be extern imported from C.
)

type Instr struct {
	Op   OpCode
	Dest IRValue
	Type Type
	Args []IRValue
	// For OpConst, the constant value is stored here.
	Const uint64
	Data  []byte
	Cmp   CmpKind
}

type Function struct {
	Name   string
	Public bool
	Blocks []*Block
}

type Block struct {
	ID     int
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
