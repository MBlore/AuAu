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
	TypeBool
	TypeVoid
	TypeFloat32
	TypeFloat64
	TypeStruct
	TypeArray
)

type Type struct {
	Kind   TypeKind
	Elem   *Type
	Len    int     // For arrays, the number of elements.
	Fields []Field // For structs, the fields of the struct.
	Name   string  // For named types, the name of the type.
}

type Field struct {
	Name string
	Type Type
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

	// OpFieldAddr computes the address of a struct field.
	OpFieldAddr

	OpPrint // temporary opcode for testing purposes, to be removed later when print can be extern imported from C.
)

type Instr struct {
	Op          OpCode    // The operation code of this instruction.
	Dest        IRValue   // The destination value of this instruction, if any.
	Type        Type      // The type of the value produced by this instruction, if any.
	Args        []IRValue // For OpLoad and OpStore, the address is stored here.
	Const       uint64    // For OpLoad and OpStore, the address is stored here.
	Data        []byte    // For OpStringConst, the string data is stored here.
	Cmp         CmpKind   // For OpCmp, the kind of comparison.
	TrueBlock   *Block    // For OpBranch, the block to jump to if the condition is true.
	FalseBlock  *Block    // For OpBranch, the block to jump to if the condition is false.
	JumpBlock   *Block    // For OpJump, the block to jump to unconditionally.
	Callee      string    // For OpCall, the function being called.
	ParamIndex  int       // ParamIndex is the flattened ABI argument index.
	FieldOffset int       // For OpFieldAddr, byte offset from the base aggregate address.
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

// PtrType creates a pointer type for the given element type.
func PtrType(elem Type) Type {
	return Type{
		Kind: TypePtr,
		Elem: &elem,
	}
}

// StringType returns the IR type representing a string.
func StringType() Type {
	return Type{
		Kind: TypeStruct,
		Name: "string",
		Fields: []Field{
			{Name: "data", Type: Type{Kind: TypePtr, Elem: &Type{Kind: TypeU8}}},
			{Name: "len", Type: Type{Kind: TypeI64}},
		},
	}
}

func (t Type) IsString() bool {
	return t.Kind == TypeStruct && t.Name == "string"
}

func (t Type) IsPtr() bool {
	return t.Kind == TypePtr
}

func (t Type) IsStruct() bool {
	return t.Kind == TypeStruct
}

func ReturnsViaHiddenPtr(t Type) bool {
	return t.IsStruct()
}

func alignUp(offset int, align int) int {
	if align <= 0 {
		panic("alignment must be positive")
	}

	rem := offset % align
	if rem == 0 {
		return offset
	}

	return offset + (align - rem)
}

// TypeAlign returns the storage alignment of a type under the compiler's
// shared aggregate layout model.
func TypeAlign(t Type) int {
	switch t.Kind {
	case TypeI8, TypeU8, TypeBool:
		return 1
	case TypeI16, TypeU16:
		return 2
	case TypeI32, TypeU32, TypeFloat32:
		return 4
	case TypeI64, TypeU64, TypePtr, TypeFloat64:
		return 8
	case TypeArray:
		if t.Elem == nil {
			panic("array type missing element type")
		}
		return TypeAlign(*t.Elem)
	case TypeStruct:
		if len(t.Fields) == 0 {
			return 1
		}

		align := 1
		for _, field := range t.Fields {
			fieldAlign := TypeAlign(field.Type)
			if fieldAlign > align {
				align = fieldAlign
			}
		}
		return align
	default:
		panic("unsupported type kind")
	}
}

// TypeSize returns the storage size of a type under the compiler's shared
// aggregate layout model.
func TypeSize(t Type) int {
	switch t.Kind {
	case TypeI8, TypeU8, TypeBool:
		return 1
	case TypeI16, TypeU16:
		return 2
	case TypeI32, TypeU32, TypeFloat32:
		return 4
	case TypeI64, TypeU64, TypePtr, TypeFloat64:
		return 8
	case TypeStruct:
		offset := 0
		for _, field := range t.Fields {
			offset = alignUp(offset, TypeAlign(field.Type))
			offset += TypeSize(field.Type)
		}
		return alignUp(offset, TypeAlign(t))
	case TypeArray:
		if t.Elem == nil {
			panic("array type missing element type")
		}
		elem := *t.Elem
		stride := alignUp(TypeSize(elem), TypeAlign(elem))
		return t.Len * stride
	default:
		panic("unsupported type kind")
	}
}

// FieldOffset returns the byte offset of a direct field within a struct type.
func FieldOffset(t Type, fieldIndex int) int {
	if !t.IsStruct() {
		panic("FieldOffset called on non-struct type")
	}

	if fieldIndex < 0 || fieldIndex >= len(t.Fields) {
		panic("field index out of bounds")
	}

	offset := 0
	for i := 0; i < fieldIndex; i++ {
		field := t.Fields[i].Type
		offset = alignUp(offset, TypeAlign(field))
		offset += TypeSize(field)
	}

	return alignUp(offset, TypeAlign(t.Fields[fieldIndex].Type))
}
