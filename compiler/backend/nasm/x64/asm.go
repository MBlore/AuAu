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
	b.WriteString("section .rdata\n")
	b.WriteString("section .text\n")

	// Compile each function in the program.
	for _, fn := range program.Functions {
		frame := buildStackFrame(fn)

		// Write the function prologue.
		b.WriteString("global " + fn.Name + "\n")
		b.WriteString(fn.Name + ":\n")
		b.WriteString("  push rbp\n")
		b.WriteString("  mov rbp, rsp\n")
		fmt.Fprintf(&b, "  sub rsp, %d\n", frame.size)

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
	case ir.OpAlloc:
		// No code needed for allocation since we reserved stack space in the prologue.
	case ir.OpConst:
		fmt.Fprintf(b, "  mov rax, %d\n", instr.Const)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpStore:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Args[0]))
	case ir.OpLoad:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
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
		fmt.Fprintf(b, "  add rax, rcx\n")
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpSub:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  sub rax, rcx\n")
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpMul:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  imul rax, rcx\n")
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpDiv:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  cqo\n")
		fmt.Fprintf(b, "  idiv rcx\n")
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpNeg:
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  neg rax\n")
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	default:
		panic(fmt.Sprintf("Unsupported OpCode: %d", instr.Op))
	}
}

// stackFrame tracks all local allocations to stack slots.
type stackFrame struct {
	// Tracking of IRValues to stack slots.
	slots map[ir.IRValue]int
	// The total size of the stack frame.
	size int
}

// buildStackFrame calculates the total stack frame size we need for the prologue of a function.
func buildStackFrame(fn *ir.Function) *stackFrame {
	frame := &stackFrame{
		slots: make(map[ir.IRValue]int),
	}

	offset := 0

	// Look for allocs throughout the blocks and instructions.
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if opCodeProducesValue(instr.Op) {
				// Move offset first to skip where RBP lives.
				offset += 8 // 64-bits per slot
				frame.slots[instr.Dest] = offset
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
