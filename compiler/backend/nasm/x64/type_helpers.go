package x64

import (
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
	default:
		panic("unsupported type kind")
	}
}

func isSigned(t ir.Type) bool {
	switch t.Kind {
	case ir.TypeI64, ir.TypeI32, ir.TypeI16, ir.TypeI8:
		return true
	case ir.TypeU64, ir.TypeU32, ir.TypeU16, ir.TypeU8, ir.TypePtr:
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
		case ir.TypeI8, ir.TypeU8:
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
		case ir.TypeI8, ir.TypeU8:
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
		case ir.TypeI8, ir.TypeU8:
			return "dl"
		default:
			panic("unsupported type kind")
		}
	}

	panic("unsupported register " + reg)
}
