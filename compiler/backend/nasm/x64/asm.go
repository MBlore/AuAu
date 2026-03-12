package x64

import (
	"bytes"
	"fmt"
	"os"

	"github.com/MBlore/AuAu/ir"
)

// This is a NASM background emitter that will take a compiled IR program and emit NASM assembly code
// compatible with the Windows x64 ABI.
func Compile(outFilename string, program *ir.IRProgram) error {
	var b bytes.Buffer

	// Write header.
	b.WriteString("default rel\n")
	b.WriteString("extern au_rt_setconsoleoutput\n")
	b.WriteString("extern printf\n")

	// Write the user externs as NASM externs.
	for _, ext := range program.Externs {
		b.WriteString("extern " + ext.Name + "\n")
	}

	b.WriteString("\nsection .rdata\n")
	b.WriteString("  fmt_int db \"%lld\", 10, 0\n")

	// Collect all string constants.
	stringLabels := map[*ir.Instr]string{}
	nextStringID := 0

	for _, fn := range program.Functions {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if instr.Op == ir.OpStringConst {
					label := fmt.Sprintf("str%d", nextStringID)

					// Record the label.
					stringLabels[instr] = label
					nextStringID++

					// Write the literal data for this string constant.
					fmt.Fprintf(&b, "  %s db ", label)
					for i, byt := range instr.Data {
						if i > 0 {
							b.WriteString(", ")
						}
						fmt.Fprintf(&b, "%d", byt)
					}

					if len(instr.Data) == 0 {
						b.WriteString("0")
					}

					b.WriteString("\n")
				}
			}
		}
	}

	b.WriteString("\nsection .text\n")

	// Compile each function in the program.
	for _, fn := range program.Functions {
		frame := buildStackFrame(fn)

		// Write function body...
		for i, block := range fn.Blocks {
			label := asmBlockLabel(fn, block)

			if i == 0 {
				if fn.Public || fn.Name == "main" {
					b.WriteString("global " + fn.Name + "\n")
				}

				b.WriteString(fn.Name + ":\n")
			}

			b.WriteString(label + ":\n")

			if i == 0 {
				// Only the first block of the func gets the prologue.
				b.WriteString("  push rbp\n")
				b.WriteString("  mov rbp, rsp\n")
				fmt.Fprintf(&b, "  sub rsp, %d\n", frame.size)

				// Main sets the codepage to support unicode.
				if fn.Name == "main" {
					b.WriteString("  call au_rt_setconsoleoutput\n")
				}
			}

			for _, instr := range block.Instrs {
				emitOpCode(&b, instr, frame, stringLabels, fn)
			}
		}

		// Function epilogue is handled by OpReturn which is always present
		// in the instructions of the body block.
	}

	err := os.WriteFile(outFilename, b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

// emitOpCode emits the assembly code for a given IR instruction based on its OpCode.
func emitOpCode(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame, stringLabels map[*ir.Instr]string, fn *ir.Function) {
	switch instr.Op {
	case ir.OpBranch:
		// Condition is in Arg0, then jump to the right block.
		fmt.Fprintf(b, "  movzx eax, byte %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  test al, al\n")
		fmt.Fprintf(b, "  jne %s\n", asmBlockLabel(fn, instr.TrueBlock))
		fmt.Fprintf(b, "  jmp %s\n", asmBlockLabel(fn, instr.FalseBlock))

	case ir.OpJump:
		fmt.Fprintf(b, "  jmp %s\n", asmBlockLabel(fn, instr.JumpBlock))

	case ir.OpCmp:
		operandType := valueType(frame, instr.Args[0])

		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  cmp %s, %s\n", regForType("rax", operandType), regForType("rcx", operandType))

		switch instr.Cmp {
		case ir.CmpEq:
			fmt.Fprintf(b, "  sete al\n")
		case ir.CmpNotEq:
			fmt.Fprintf(b, "  setne al\n")
		case ir.CmpLt:
			if isSigned(operandType) {
				fmt.Fprintf(b, "  setl al\n")
			} else {
				fmt.Fprintf(b, "  setb al\n")
			}
		case ir.CmpLtEq:
			if isSigned(operandType) {
				fmt.Fprintf(b, "  setle al\n")
			} else {
				fmt.Fprintf(b, "  setbe al\n")
			}
		case ir.CmpGt:
			if isSigned(operandType) {
				fmt.Fprintf(b, "  setg al\n")
			} else {
				fmt.Fprintf(b, "  seta al\n")
			}
		case ir.CmpGtEq:
			if isSigned(operandType) {
				fmt.Fprintf(b, "  setge al\n")
			} else {
				fmt.Fprintf(b, "  setae al\n")
			}
		default:
			panic(fmt.Sprintf("unsupported cmp kind: %d", instr.Cmp))
		}

		fmt.Fprintf(b, "  movzx eax, al\n") // Zero-extend the result to 64 bits in RAX.
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpStringConst:
		label := stringLabels[instr]

		fmt.Fprintf(b, "  ; String constant: %s\n", label)
		fmt.Fprintf(b, "  mov rax, %s\n", label)
		fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 0)) // ptr

		fmt.Fprintf(b, "  mov rax, %d\n", len(instr.Data))
		fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 8)) // len
		fmt.Fprintf(b, "  ; End string constant: %s\n", label)
	case ir.OpPrint:
		// Call printf with the value in RAX.
		fmt.Fprintf(b, "  mov rcx, fmt_int\n")
		fmt.Fprintf(b, "  mov rdx, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  call printf\n")
	case ir.OpAlloc:
		// No code needed for allocation since we reserved stack space in the prologue.
	case ir.OpConst:
		fmt.Fprintf(b, "  mov rax, %d\n", instr.Const)
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpStore:
		// Store a value from one stack slot to another, via RAX.
		addrType := valueType(frame, instr.Args[0])
		if addrType.Kind != ir.TypePtr || addrType.Elem == nil {
			panic("invalid store operation: store destination is not a pointer")
		}

		storeType := *addrType.Elem

		// Handle string save (ptr + len).
		if storeType.Kind == ir.TypeString {
			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[1], 0))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Args[0], 0))

			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[1], 8))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Args[0], 8))
			break
		}

		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  mov %s, %s\n", sizedMem(slot(frame, instr.Args[0]), storeType), regForType("rax", storeType))
	case ir.OpLoad:
		// Handle strings (ptr + len copy).
		if instr.Type.Kind == ir.TypeString {
			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[0], 0))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 0))
			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[0], 8))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 8))
			break
		}

		// Moves a value from one stack slot to another, via RAX.
		emitLoadIntoRAX(b, instr, frame)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpReturn:
		// Write the function epilogue and return.
		// Single return value goes in rax.
		// Multiple return values require allocated memory from the caller with a pointer
		// passed as an arguement that we use to store a return value in.
		if len(instr.Args) > 0 {
			fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		}
		fmt.Fprintf(b, "  mov rsp, rbp\n")
		fmt.Fprintf(b, "  pop rbp\n")
		fmt.Fprintf(b, "  ret\n")
	case ir.OpAdd:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  add %s, %s\n", regForType("rax", instr.Type), regForType("rcx", instr.Type))
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpSub:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  sub %s, %s\n", regForType("rax", instr.Type), regForType("rcx", instr.Type))
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpMul:
		bs := bitSize(instr.Type)
		signed := isSigned(instr.Type)

		switch bs {
		case 64:
			fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
			if signed {
				fmt.Fprintf(b, "  imul rcx\n")
			} else {
				fmt.Fprintf(b, "  mul rcx\n")
			}

		case 32, 16, 8:
			fmt.Fprintf(b, "  mov eax, %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov ecx, %s\n", slot(frame, instr.Args[1]))
			if signed {
				fmt.Fprintf(b, "  imul ecx\n")
			} else {
				fmt.Fprintf(b, "  mul ecx\n")
			}

		default:
			panic(fmt.Sprintf("unsupported mul width: %d", bs))
		}

		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))

	case ir.OpDiv:
		bs := bitSize(instr.Type)
		signed := isSigned(instr.Type)

		switch bs {
		case 64:
			fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))

			if signed {
				fmt.Fprintf(b, "  cqo\n")
				fmt.Fprintf(b, "  idiv rcx\n")
			} else {
				fmt.Fprintf(b, "  xor rdx, rdx\n")
				fmt.Fprintf(b, "  div rcx\n")
			}

		case 32, 16, 8:
			fmt.Fprintf(b, "  mov eax, %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov ecx, %s\n", slot(frame, instr.Args[1]))

			if signed {
				fmt.Fprintf(b, "  cdq\n")
				fmt.Fprintf(b, "  idiv ecx\n")
			} else {
				fmt.Fprintf(b, "  xor edx, edx\n")
				fmt.Fprintf(b, "  div ecx\n")
			}

		default:
			panic(fmt.Sprintf("unsupported div width: %d", bs))
		}

		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))

	case ir.OpNeg:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  neg %s\n", regForType("rax", instr.Type))
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	default:
		panic(fmt.Sprintf("Unsupported OpCode: %d", instr.Op))
	}
}

// emitCanonicalizeRAX emits additional instructions to ensure that the value in RAX is properly sign-extended or zero-extended.
// This is because our stack slots are 64-bits and we do ops on smaller types leaving RAX garbage.
// So this extra instruction cleans up RAX.
func emitCanonicalizeRAX(b *bytes.Buffer, instr *ir.Instr) {
	switch instr.Type.Kind {
	case ir.TypeI64, ir.TypeU64, ir.TypePtr:
		// Do nothing since the add instruction already modifies rax in place for 64-bit types.
	case ir.TypeI8:
		fmt.Fprintf(b, "  movsx rax, al\n")
	case ir.TypeU8:
		fmt.Fprintf(b, "  movzx eax, al\n")
	case ir.TypeI16:
		fmt.Fprintf(b, "  movsx rax, ax\n")
	case ir.TypeU16:
		fmt.Fprintf(b, "  movzx eax, ax\n")
	case ir.TypeI32:
		fmt.Fprintf(b, "  movsxd rax, eax\n")
	case ir.TypeU32:
		fmt.Fprintf(b, "  mov eax, eax\n")
	case ir.TypeBool:
		fmt.Fprintf(b, "  movzx eax, al\n")
	default:
		panic(fmt.Sprintf("unsupported type: %d", instr.Type.Kind))
	}
}

// emitLoadIntoRAX emits the appropriate instructions to load a value from a stack slot into RAX based on the type of the value being loaded.
func emitLoadIntoRAX(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame) {
	switch instr.Type.Kind {
	case ir.TypeI64, ir.TypeU64, ir.TypePtr:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
	case ir.TypeI32:
		fmt.Fprintf(b, "  mov eax, dword %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  movsxd rax, eax\n")
	case ir.TypeU32:
		fmt.Fprintf(b, "  mov eax, dword %s\n", slot(frame, instr.Args[0]))
	case ir.TypeI16:
		fmt.Fprintf(b, "  movsx rax, word %s\n", slot(frame, instr.Args[0]))
	case ir.TypeU16:
		fmt.Fprintf(b, "  movzx eax, word %s\n", slot(frame, instr.Args[0]))
	case ir.TypeI8:
		fmt.Fprintf(b, "  movsx rax, byte %s\n", slot(frame, instr.Args[0]))
	case ir.TypeU8, ir.TypeBool:
		fmt.Fprintf(b, "  movzx eax, byte %s\n", slot(frame, instr.Args[0]))
	default:
		panic(fmt.Sprintf("unsupported load type: %d", instr.Type.Kind))
	}
}

// stackFrame tracks all local allocations to stack slots.
type stackFrame struct {
	// Tracking of IRValues to stack slots.
	slots map[ir.IRValue]int
	// Tracking of IRValues to their types.
	types map[ir.IRValue]ir.Type
	// The total size of the stack frame.
	size int
}

// buildStackFrame calculates the total stack frame size we need for the prologue of a function.
func buildStackFrame(fn *ir.Function) *stackFrame {
	frame := &stackFrame{
		slots: make(map[ir.IRValue]int),
		types: make(map[ir.IRValue]ir.Type),
	}

	offset := 0

	// Look for allocs throughout the blocks and instructions.
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if opCodeProducesValue(instr.Op) {
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
	case ir.TypeString:
		return 16
	case ir.TypeI8, ir.TypeI16, ir.TypeI32, ir.TypeI64,
		ir.TypeU8, ir.TypeU16, ir.TypeU32, ir.TypeU64,
		ir.TypePtr, ir.TypeBool:
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
		ir.OpStringConst:
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

// asmBlockLabel generates a unique label for a given IR block within a function for use in assembly code.
func asmBlockLabel(fn *ir.Function, block *ir.Block) string {
	if block.Name != "" {
		return fmt.Sprintf("%s_block%d_%s", fn.Name, block.ID, block.Name)
	}
	return fmt.Sprintf("%s_block%d", fn.Name, block.ID)
}
