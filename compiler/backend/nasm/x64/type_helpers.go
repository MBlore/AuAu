package x64

import (
	"fmt"

	"github.com/MBlore/AuAu/ir"
)

// bitSize returns the size in bits of a given IR type.
func bitSize(t ir.Type) int {
	switch t.Kind {
	case ir.TypeI64:
		return 64
	case ir.TypeI32:
		return 32
	case ir.TypeI16:
		return 16
	case ir.TypeI8:
		return 8
	case ir.TypeU64:
		return 64
	case ir.TypeU32:
		return 32
	case ir.TypeU16:
		return 16
	case ir.TypeU8:
		return 8
	case ir.TypePtr:
		return 64
	case ir.TypeBool:
		return 8
	case ir.TypeFloat32:
		return 32
	case ir.TypeFloat64:
		return 64
	default:
		panic("unsupported type kind")
	}
}

func isSigned(t ir.Type) bool {
	switch t.Kind {
	case ir.TypeI64, ir.TypeI32, ir.TypeI16, ir.TypeI8:
		return true
	case ir.TypeU64, ir.TypeU32, ir.TypeU16, ir.TypeU8, ir.TypePtr, ir.TypeBool:
		return false
	default:
		panic("unsupported type kind")
	}
}

func regForType(reg string, t ir.Type) string {
	switch reg {
	case "rax":
		switch t.Kind {
		case ir.TypeI64, ir.TypeU64, ir.TypePtr:
			return "rax"
		case ir.TypeI32, ir.TypeU32:
			return "eax"
		case ir.TypeI16, ir.TypeU16:
			return "ax"
		case ir.TypeI8, ir.TypeU8, ir.TypeBool:
			return "al"
		default:
			panic("unsupported type kind")
		}
	case "rcx":
		switch t.Kind {
		case ir.TypeI64, ir.TypeU64, ir.TypePtr:
			return "rcx"
		case ir.TypeI32, ir.TypeU32:
			return "ecx"
		case ir.TypeI16, ir.TypeU16:
			return "cx"
		case ir.TypeI8, ir.TypeU8, ir.TypeBool:
			return "cl"
		default:
			panic("unsupported type kind")
		}
	case "rdx":
		switch t.Kind {
		case ir.TypeI64, ir.TypeU64, ir.TypePtr:
			return "rdx"
		case ir.TypeI32, ir.TypeU32:
			return "edx"
		case ir.TypeI16, ir.TypeU16:
			return "dx"
		case ir.TypeI8, ir.TypeU8, ir.TypeBool:
			return "dl"
		default:
			panic("unsupported type kind")
		}
	}

	panic("unsupported register " + reg)
}

// sizedMem returns the appropriate memory operand string for a given IR type, including the correct size prefix (byte, word, dword, qword).
// This is NASM syntax for memory operands, e.g. "byte [rbp-8]" or "dword [rbp-16]".
func sizedMem(operand string, t ir.Type) string {
	switch bitSize(t) {
	case 8:
		return "byte " + operand
	case 16:
		return "word " + operand
	case 32:
		return "dword " + operand
	case 64:
		return "qword " + operand
	default:
		panic(fmt.Sprintf("unsupported memory width: %d", bitSize(t)))
	}
}
