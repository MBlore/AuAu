package x64

import (
	"bytes"
	"fmt"

	"github.com/MBlore/AuAu/ir"
)

// callArgs helps track the type and location of each argument for a call instruction.
type callArg struct {
	typ       ir.Type // The type of the argument, used for determining how to pass it.
	slot      string  // The memory location or register for this argument.
	memSize   string  // qword, dword, etc. for memory arguments
	isAddress bool    // If true, it means the slot is an address that needs to be dereferenced when loading the argument.
}

func emitCall(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame) {
	args := flattenCallArgs(instr, frame)

	// Calculate how many bytes we need to push on the stack for the arguments.
	stackArgCount := 0
	if len(args) > 4 {
		stackArgCount = len(args) - 4
	}

	stackBytes := stackArgCount * 8

	// Keep the stack size aligned.
	if stackBytes%16 != 0 {
		stackBytes += 8
	}

	if stackArgCount > 0 {
		// Move the stack pointer to make room for the args.
		fmt.Fprintf(b, "  sub rsp, %d\n", stackBytes)
	}

	emitCallArgs(b, args)

	fmt.Fprintf(b, "  call %s\n", instr.Callee)

	// Undo the stack pointer adjustment after the call.
	if stackBytes > 0 {
		fmt.Fprintf(b, "  add rsp, %d\n", stackBytes)
	}

	emitCallReturn(b, instr, frame)
}

// emitCallArgs puts arguements on to registers/stack ready for a function call.
func emitCallArgs(b *bytes.Buffer, args []callArg) {
	// Put the arguements on to the stack in reverse order (right to left).
	stackOffset := 0

	// First, the stack arguements if any.
	for i := len(args) - 1; i >= 4; i-- {
		arg := args[i]

		// If the argument is an address, we need to load the address itself onto the stack, not the value at that address.
		if arg.isAddress {
			fmt.Fprintf(b, "  lea rax, %s\n", arg.slot)
			fmt.Fprintf(b, "  mov qword [rsp+%d], rax\n", stackOffset)
			stackOffset += 8
			continue
		}

		switch arg.typ.Kind {
		case ir.TypeFloat32:
			fmt.Fprintf(b, "  mov eax, dword %s\n", arg.slot)
			fmt.Fprintf(b, "  mov dword [rsp + %d], eax\n", stackOffset)
		case ir.TypeFloat64:
			fmt.Fprintf(b, "  mov rax, qword %s\n", arg.slot)
			fmt.Fprintf(b, "  mov qword [rsp + %d], rax\n", stackOffset)
		default:
			emitLoadCallStackArg(b, arg, stackOffset)
		}

		stackOffset += 8
	}

	// Now the register arguments.
	limit := min(len(args), 4)

	for i := 0; i < limit; i++ {
		arg := args[i]

		// If the argument is an address, we need to load the address itself into the register, not the value at that address.
		if arg.isAddress {
			fmt.Fprintf(b, "  lea %s, %s\n", callIntArgReg(i), arg.slot)
			continue
		}

		switch arg.typ.Kind {
		case ir.TypeFloat32:
			fmt.Fprintf(b, "  movss %s, dword %s\n", callFloatArgReg(i), arg.slot)
		case ir.TypeFloat64:
			fmt.Fprintf(b, "  movsd %s, qword %s\n", callFloatArgReg(i), arg.slot)
		default:
			emitLoadCallIntArg(b, callIntArgReg(i), arg)
		}
	}
}

// emitLoadCallStackArg emits the assembly code to load a call argument from the stack into rax, and then move it to the appropriate stack slot for the call.
func emitLoadCallStackArg(b *bytes.Buffer, arg callArg, stackOffset int) {
	switch arg.typ.Kind {
	case ir.TypeI64, ir.TypeU64, ir.TypePtr:
		fmt.Fprintf(b, "  mov rax, qword %s\n", arg.slot)

	case ir.TypeI32:
		fmt.Fprintf(b, "  movsxd rax, dword %s\n", arg.slot)

	case ir.TypeU32:
		fmt.Fprintf(b, "  mov eax, dword %s\n", arg.slot)

	case ir.TypeI16:
		fmt.Fprintf(b, "  movsx rax, word %s\n", arg.slot)

	case ir.TypeU16:
		fmt.Fprintf(b, "  movzx eax, word %s\n", arg.slot)

	case ir.TypeI8:
		fmt.Fprintf(b, "  movsx rax, byte %s\n", arg.slot)

	case ir.TypeU8, ir.TypeBool:
		fmt.Fprintf(b, "  movzx eax, byte %s\n", arg.slot)

	default:
		panic(fmt.Sprintf("unsupported stack call arg type: %d", arg.typ.Kind))
	}

	fmt.Fprintf(b, "  mov qword [rsp+%d], rax\n", stackOffset)
}

// emitLoadCallIntArg emits the right assembly code for varying int types.
func emitLoadCallIntArg(b *bytes.Buffer, reg string, arg callArg) {
	switch arg.typ.Kind {
	case ir.TypeI64, ir.TypeU64, ir.TypePtr:
		fmt.Fprintf(b, "  mov %s, qword %s\n", reg, arg.slot)

	case ir.TypeI32:
		if reg == "r8" {
			fmt.Fprintf(b, "  movsxd r8, dword %s\n", arg.slot)
		} else if reg == "r9" {
			fmt.Fprintf(b, "  movsxd r9, dword %s\n", arg.slot)
		} else {
			reg32 := reg[:1] + "x"

			if reg == "rcx" {
				reg32 = "ecx"
			} else if reg == "rdx" {
				reg32 = "edx"
			}

			fmt.Fprintf(b, "  mov %s, dword %s\n", reg32, arg.slot)
			fmt.Fprintf(b, "  movsxd %s, %s\n", reg, reg32)
		}

	case ir.TypeU32:
		switch reg {
		case "rcx":
			fmt.Fprintf(b, "  mov ecx, dword %s\n", arg.slot)
		case "rdx":
			fmt.Fprintf(b, "  mov edx, dword %s\n", arg.slot)
		case "r8":
			fmt.Fprintf(b, "  mov r8d, dword %s\n", arg.slot)
		case "r9":
			fmt.Fprintf(b, "  mov r9d, dword %s\n", arg.slot)
		}

	case ir.TypeI16:
		switch reg {
		case "rcx":
			fmt.Fprintf(b, "  movsx rcx, word %s\n", arg.slot)
		case "rdx":
			fmt.Fprintf(b, "  movsx rdx, word %s\n", arg.slot)
		case "r8":
			fmt.Fprintf(b, "  movsx r8, word %s\n", arg.slot)
		case "r9":
			fmt.Fprintf(b, "  movsx r9, word %s\n", arg.slot)
		}

	case ir.TypeU16:
		switch reg {
		case "rcx":
			fmt.Fprintf(b, "  movzx ecx, word %s\n", arg.slot)
		case "rdx":
			fmt.Fprintf(b, "  movzx edx, word %s\n", arg.slot)
		case "r8":
			fmt.Fprintf(b, "  movzx r8d, word %s\n", arg.slot)
		case "r9":
			fmt.Fprintf(b, "  movzx r9d, word %s\n", arg.slot)
		}

	case ir.TypeI8:
		switch reg {
		case "rcx":
			fmt.Fprintf(b, "  movsx rcx, byte %s\n", arg.slot)
		case "rdx":
			fmt.Fprintf(b, "  movsx rdx, byte %s\n", arg.slot)
		case "r8":
			fmt.Fprintf(b, "  movsx r8, byte %s\n", arg.slot)
		case "r9":
			fmt.Fprintf(b, "  movsx r9, byte %s\n", arg.slot)
		}

	case ir.TypeU8, ir.TypeBool:
		switch reg {
		case "rcx":
			fmt.Fprintf(b, "  movzx ecx, byte %s\n", arg.slot)
		case "rdx":
			fmt.Fprintf(b, "  movzx edx, byte %s\n", arg.slot)
		case "r8":
			fmt.Fprintf(b, "  movzx r8d, byte %s\n", arg.slot)
		case "r9":
			fmt.Fprintf(b, "  movzx r9d, byte %s\n", arg.slot)
		}

	default:
		panic(fmt.Sprintf("unsupported call arg type: %d", arg.typ.Kind))
	}
}

// emitCallReturn emits the assembly code to move the return value of a call instruction into the appropriate destination based on the return type.
func emitCallReturn(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame) {
	if instr.Type.Kind == ir.TypeVoid || ir.ReturnsViaHiddenPtr(instr.Type) {
		return
	}

	switch instr.Type.Kind {
	case ir.TypeFloat32:
		fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
	case ir.TypeFloat64:
		fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
	default:
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	}
}

// flattenCallArgs takes a call instruction and flattens its arguments into a slice of callArg
// structs that describe how to pass each argument according to the x64 calling convention.
func flattenCallArgs(instr *ir.Instr, frame *stackFrame) []callArg {
	args := make([]callArg, 0, len(instr.Args))

	// If the function returns a struct via a hidden pointer, we need to pass that pointer
	// as the first argument.
	if ir.ReturnsViaHiddenPtr(instr.Type) {
		args = append(args, callArg{
			typ:       ir.PtrType(instr.Type),
			slot:      slot(frame, instr.Dest),
			memSize:   "qword",
			isAddress: true,
		})
	}

	// Now flatten the regular arguments.
	for _, arg := range instr.Args {
		t := valueType(frame, arg)
		args = append(args, flattenCallValue(frame, arg, t, 0, false)...)
	}

	return args
}

// flattenCallValue takes an IRValue that is being passed as an argument to a call instruction,
// and flattens it into one or more callArg structs that describe how to pass the value according
// to the x64 calling convention. This includes handling struct values by flattening their fields
// into separate arguments.
func flattenCallValue(frame *stackFrame, value ir.IRValue, t ir.Type, fieldOffset int, isField bool) []callArg {
	if t.Kind != ir.TypeStruct {
		// For non-struct types, we can pass the value directly.
		slotRef := slot(frame, value)
		if isField {
			slotRef = slotField(frame, value, fieldOffset)
		}

		return []callArg{{
			typ:     t,
			slot:    slotRef,
			memSize: memPrefix(t),
		}}
	}

	// For struct types, we need to flatten each field into a separate argument.
	args := []callArg{}

	for i, field := range t.Fields {
		offset := fieldOffset + ir.FieldOffset(t, i)
		args = append(args, flattenCallValue(frame, value, field.Type, offset, true)...)
	}

	return args
}

func memPrefix(t ir.Type) string {
	switch bitSize(t) {
	case 8:
		return "byte"
	case 16:
		return "word"
	case 32:
		return "dword"
	case 64:
		return "qword"
	default:
		panic(fmt.Sprintf("unsupported memory width: %d", bitSize(t)))
	}
}

func callFloatArgReg(index int) string {
	switch index {
	case 0:
		return "xmm0"
	case 1:
		return "xmm1"
	case 2:
		return "xmm2"
	case 3:
		return "xmm3"
	default:
		panic(fmt.Sprintf("unsupported float argument register index: %d", index))
	}
}

func callIntArgReg(index int) string {
	switch index {
	case 0:
		return "rcx"
	case 1:
		return "rdx"
	case 2:
		return "r8"
	case 3:
		return "r9"
	default:
		panic(fmt.Sprintf("unsupported integer argument register index: %d", index))
	}
}
