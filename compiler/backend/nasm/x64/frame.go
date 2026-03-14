package x64

import (
	"fmt"

	"github.com/MBlore/AuAu/ir"
)

// stackFrame tracks all local allocations to stack slots.
type stackFrame struct {
	// Tracking of IRValues to stack slots.
	slots map[ir.IRValue]int
	// Tracking of OpAlloc IRValues to the base offset of their pointee storage.
	allocBases map[ir.IRValue]int
	// Tracking of IRValues to their types.
	types map[ir.IRValue]ir.Type
	// The total size of the stack frame.
	size int
}

// buildStackFrame calculates the total stack frame size we need for the prologue of a function.
func buildStackFrame(fn *ir.Function) *stackFrame {
	frame := &stackFrame{
		slots:      make(map[ir.IRValue]int),
		allocBases: make(map[ir.IRValue]int),
		types:      make(map[ir.IRValue]ir.Type),
	}

	offset := 0

	// Look for allocs throughout the blocks and instructions.
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if opCodeProducesValue(instr.Op) {
				if instr.Op == ir.OpAlloc {
					if instr.Type.Kind != ir.TypePtr || instr.Type.Elem == nil {
						panic("alloc must produce a pointer type with an element type")
					}

					// Reserve stack space for the pointee storage first.
					offset += stackSize(*instr.Type.Elem)
					frame.allocBases[instr.Dest] = offset

					// Reserve a separate slot to hold the pointer value itself.
					offset += 8
					frame.slots[instr.Dest] = offset
					frame.types[instr.Dest] = instr.Type
					continue
				}

				// Move offset first to skip where RBP lives.
				offset += stackSize(instr.Type)

				// Track the slot and type for this IRValue.
				frame.slots[instr.Dest] = offset
				frame.types[instr.Dest] = instr.Type
			}
		}
	}

	// Total stack size is aligned to 16 bytes + 32 bytes for shadow space.
	frame.size = align16(offset + 32)
	return frame
}

// stackSize returns the size in bytes of the given IR type when stored on the stack.
func stackSize(t ir.Type) int {
	switch t.Kind {
	case ir.TypeStruct:
		size := 0

		for _, field := range t.Fields {
			size += stackSize(field.Type)
		}

		return size

	case ir.TypeArray:
		if t.Elem == nil {
			panic("array type missing element type")
		}

		return t.Len * stackSize(*t.Elem)

	case ir.TypeI8, ir.TypeI16, ir.TypeI32, ir.TypeI64,
		ir.TypeU8, ir.TypeU16, ir.TypeU32, ir.TypeU64,
		ir.TypePtr, ir.TypeBool, ir.TypeFloat32, ir.TypeFloat64:
		return 8

	default:
		panic("unsupported type kind")
	}
}

// opCodeProducesValue returns true if the given OpCode produces a value that needs
// to be stored in a stack slot.
func opCodeProducesValue(op ir.OpCode) bool {
	switch op {
	case ir.OpAdd,
		ir.OpSub,
		ir.OpMul,
		ir.OpDiv,
		ir.OpConst,
		ir.OpLoad,
		ir.OpAlloc,
		ir.OpNeg,
		ir.OpCmp,
		ir.OpStringConst,
		ir.OpCall,
		ir.OpParam,
		ir.OpFieldAddr:
		return true
	default:
		return false
	}
}

// align16 rounds the given size up to the nearest multiple of 16 bytes,
// which is required for stack alignment in the Windows x64 ABI.
func align16(size int) int {
	if size%16 == 0 {
		return size
	}
	return size + (16 - (size % 16))
}

// slot returns the assembly operand for the given IRValue based on its assigned stack slot.
func slot(frame *stackFrame, value ir.IRValue) string {
	offset := frame.slots[value]
	return fmt.Sprintf("[rbp-%d]", offset)
}

// slotField returns the assembly operand for a specific field within a struct stored in a stack slot.
func slotField(frame *stackFrame, value ir.IRValue, fieldOffset int) string {
	offset := frame.slots[value]
	return fmt.Sprintf("[rbp-%d]", offset-fieldOffset)
}

// valueType returns the type of the given IRValue from the stack frame's type tracking.
func valueType(frame *stackFrame, value ir.IRValue) ir.Type {
	t, ok := frame.types[value]
	if !ok {
		panic(fmt.Sprintf("missing type for IR value %d", value))
	}

	return t
}
