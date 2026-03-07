package ir

type IRType int

const (
	IRTypeInvalid IRType = iota
	IRTypeI64
	IRTypeU64
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
	OpRet
	OpAlloc
)

type Instr struct {
	Op   OpCode
	Dest IRValue
	Args []IRValue
	// For OpConst, the constant value is stored here.
	Const int64
}

type Function struct {
	Name   string
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
