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
	b.WriteString("extern SetConsoleOutputCP\n")
	b.WriteString("extern printf\n")
	b.WriteString("section .rdata\n")
	b.WriteString("fmt_int db \"%lld\", 10, 0\n")
	b.WriteString("section .text\n")

	// Compile each function in the program.
	for _, fn := range program.Functions {
		frame := buildStackFrame(fn)

		// Write the function prologue.
		if fn.Public || fn.Name == "main" {
			b.WriteString("global " + fn.Name + "\n")
		}
		b.WriteString(fn.Name + ":\n")
		b.WriteString("  push rbp\n")
		b.WriteString("  mov rbp, rsp\n")
		fmt.Fprintf(&b, "  sub rsp, %d\n", frame.size)

		if fn.Name == "main" {
			// Call SetConsoleOutputCP at the start of main to set the console code page to UTF-8.
			b.WriteString("  mov ecx, 65001 ; CP_UTF8\n")
			b.WriteString("  call SetConsoleOutputCP\n")

		}

		// Write function body...
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				emitOpCode(&b, instr, frame)
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
func emitOpCode(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame) {
	switch instr.Op {
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
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  mov %s, %s\n", sizedMem(slot(frame, instr.Args[0]), storeType), regForType("rax", storeType))
	case ir.OpLoad:
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
	case ir.TypeU8:
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
				offset += 8 // 64-bits per slot

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
		ir.OpNeg:
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

// valueType returns the type of the given IRValue from the stack frame's type tracking.
func valueType(frame *stackFrame, value ir.IRValue) ir.Type {
	t, ok := frame.types[value]
	if !ok {
		panic(fmt.Sprintf("missing type for IR value %d", value))
	}

	return t
}
