package x64

import (
	"bytes"
	"fmt"

	"github.com/MBlore/AuAu/ir"
)

// emitOpCode emits the assembly code for a given IR instruction based on its OpCode.
func emitOpCode(b *bytes.Buffer, instr *ir.Instr, frame *stackFrame, stringLabels map[*ir.Instr]string, fn *ir.Function) {
	switch instr.Op {
	case ir.OpCall:
		emitCall(b, instr, frame)

	case ir.OpParam:
		// Move the parameter from the appropriate register to the stack slot for this parameter.
		switch instr.Type.Kind {
		case ir.TypeFloat32:
			fmt.Fprintf(b, "  movss dword %s, %s\n", slot(frame, instr.Dest), callFloatArgReg(instr.ParamIndex))
		case ir.TypeFloat64:
			fmt.Fprintf(b, "  movsd qword %s, %s\n", slot(frame, instr.Dest), callFloatArgReg(instr.ParamIndex))
		default:
			fmt.Fprintf(b, "  mov qword %s, %s\n", slot(frame, instr.Dest), callIntArgReg(instr.ParamIndex))
		}

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

		// Handle floats.
		if operandType.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movss xmm1, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  ucomiss xmm0, xmm1\n")
			switch instr.Cmp {
			case ir.CmpEq:
				fmt.Fprintf(b, "  sete al\n")
			case ir.CmpNotEq:
				fmt.Fprintf(b, "  setne al\n")
			case ir.CmpLt:
				fmt.Fprintf(b, "  setb al\n")
			case ir.CmpLtEq:
				fmt.Fprintf(b, "  setbe al\n")
			case ir.CmpGt:
				fmt.Fprintf(b, "  seta al\n")
			case ir.CmpGtEq:
				fmt.Fprintf(b, "  setae al\n")
			default:
				panic(fmt.Sprintf("unsupported cmp kind: %d", instr.Cmp))
			}

			fmt.Fprintf(b, "  movzx eax, al\n") // Zero-extend the result to 64 bits in RAX.
			fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
			return
		}

		if operandType.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movsd xmm1, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  ucomisd xmm0, xmm1\n")

			switch instr.Cmp {
			case ir.CmpEq:
				fmt.Fprintf(b, "  sete al\n")
			case ir.CmpNotEq:
				fmt.Fprintf(b, "  setne al\n")
			case ir.CmpLt:
				fmt.Fprintf(b, "  setb al\n")
			case ir.CmpLtEq:
				fmt.Fprintf(b, "  setbe al\n")
			case ir.CmpGt:
				fmt.Fprintf(b, "  seta al\n")
			case ir.CmpGtEq:
				fmt.Fprintf(b, "  setae al\n")
			default:
				panic(fmt.Sprintf("unsupported cmp kind: %d", instr.Cmp))
			}

			fmt.Fprintf(b, "  movzx eax, al\n") // Zero-extend the result to 64 bits in RAX.
			fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
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
		// Floats are handled differently as they are float bits.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  mov eax, %d\n", uint32(instr.Const))
			fmt.Fprintf(b, "  mov dword %s, eax\n", slot(frame, instr.Dest))
			return
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  mov rax, %d\n", instr.Const)
			fmt.Fprintf(b, "  mov qword %s, rax\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
		fmt.Fprintf(b, "  mov rax, %d\n", instr.Const)
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))

	case ir.OpStore:
		// Store the source value into the memory referenced by the destination pointer.
		addrType := valueType(frame, instr.Args[0])
		if addrType.Kind != ir.TypePtr || addrType.Elem == nil {
			panic("invalid store operation: store destination is not a pointer")
		}

		storeType := *addrType.Elem

		// Handle floats.
		if storeType.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  mov eax, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  mov dword %s, eax\n", slot(frame, instr.Args[0]))
			break
		}

		if storeType.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  mov rax, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  mov qword %s, rax\n", slot(frame, instr.Args[0]))
			break
		}

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
		// Handle floats.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  mov eax, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov dword %s, eax\n", slot(frame, instr.Dest))
			break
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  mov rax, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  mov qword %s, rax\n", slot(frame, instr.Dest))
			break
		}

		// Handle strings (ptr + len copy).
		if instr.Type.Kind == ir.TypeString {
			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[0], 0))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 0))
			fmt.Fprintf(b, "  mov rax, %s\n", slotField(frame, instr.Args[0], 8))
			fmt.Fprintf(b, "  mov %s, rax\n", slotField(frame, instr.Dest, 8))
			break
		}

		// Handle ints.
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
		// Handle floats.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movss xmm1, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  addss xmm0, xmm1\n")
			fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movsd xmm1, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  addsd xmm0, xmm1\n")
			fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  add %s, %s\n", regForType("rax", instr.Type), regForType("rcx", instr.Type))
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpSub:
		// Handle floats.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movss xmm1, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  subss xmm0, xmm1\n")
			fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movsd xmm1, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  subsd xmm0, xmm1\n")
			fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
		fmt.Fprintf(b, "  mov rax, %s\n", slot(frame, instr.Args[0]))
		fmt.Fprintf(b, "  mov rcx, %s\n", slot(frame, instr.Args[1]))
		fmt.Fprintf(b, "  sub %s, %s\n", regForType("rax", instr.Type), regForType("rcx", instr.Type))
		emitCanonicalizeRAX(b, instr)
		fmt.Fprintf(b, "  mov %s, rax\n", slot(frame, instr.Dest))
	case ir.OpMul:
		// Handle floats.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movss xmm1, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  mulss xmm0, xmm1\n")
			fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movsd xmm1, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  mulsd xmm0, xmm1\n")
			fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
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
		// Handle floats.
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movss xmm1, dword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  divss xmm0, xmm1\n")
			fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  movsd xmm1, qword %s\n", slot(frame, instr.Args[1]))
			fmt.Fprintf(b, "  divsd xmm0, xmm1\n")
			fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
			return
		}

		// Integers.
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
		// Handle floats (we can use the NASM rel...).
		if instr.Type.Kind == ir.TypeFloat32 {
			fmt.Fprintf(b, "  movss xmm0, dword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  xorps xmm0, [rel f32_sign_mask]\n")
			fmt.Fprintf(b, "  movss dword %s, xmm0\n", slot(frame, instr.Dest))
			break
		}

		if instr.Type.Kind == ir.TypeFloat64 {
			fmt.Fprintf(b, "  movsd xmm0, qword %s\n", slot(frame, instr.Args[0]))
			fmt.Fprintf(b, "  xorpd xmm0, [rel f64_sign_mask]\n")
			fmt.Fprintf(b, "  movsd qword %s, xmm0\n", slot(frame, instr.Dest))
			break
		}

		// Integers.
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
