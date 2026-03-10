package ir

import (
	"fmt"
	"strings"
)

func (p *IRProgram) String() string {
	var b strings.Builder

	for _, fn := range p.Functions {
		fmt.Fprintf(&b, "func %s:\n", fn.Name)

		for _, block := range fn.Blocks {
			fmt.Fprintf(&b, "block%d:\n", block.ID)
			for _, instr := range block.Instrs {
				fmt.Fprintf(&b, "  %s\n", InstrToString(instr))
			}
		}
	}

	return b.String()
}

// InstrToString converts an IR instruction to a human-readable string representation.
func InstrToString(instr *Instr) string {
	switch instr.Op {
	case OpPrint:
		return fmt.Sprintf("print %s", printValue(instr.Args[0]))
	case OpConst:
		return fmt.Sprintf("%s = const %d",
			printTypedValue(instr.Dest, instr.Type),
			instr.Const,
		)
	case OpLoad:
		return fmt.Sprintf("%s = load %s",
			printTypedValue(instr.Dest, instr.Type),
			printValue(instr.Args[0]),
		)
	case OpStore:
		return fmt.Sprintf("store %s, %s",
			printValue(instr.Args[0]),
			printValue(instr.Args[1]),
		)
	case OpAdd, OpSub, OpMul, OpDiv:
		return fmt.Sprintf("%s = %s %s, %s",
			printTypedValue(instr.Dest, instr.Type),
			opToString(instr.Op),
			printValue(instr.Args[0]),
			printValue(instr.Args[1]),
		)
	case OpNeg:
		return fmt.Sprintf("%s = neg %s",
			printTypedValue(instr.Dest, instr.Type),
			printValue(instr.Args[0]),
		)
	case OpAlloc:
		return fmt.Sprintf("%s = alloc",
			printTypedValue(instr.Dest, instr.Type),
		)
	case OpReturn:
		if len(instr.Args) == 0 {
			return "ret"
		}

		return fmt.Sprintf("ret %s", printValue(instr.Args[0]))
	default:
		return fmt.Sprintf("%s = <unknown op %d>",
			printTypedValue(instr.Dest, instr.Type),
			instr.Op,
		)
	}
}

func printValue(v IRValue) string {
	return fmt.Sprintf("t%d", v)
}

func printTypedValue(v IRValue, tp Type) string {
	return fmt.Sprintf("%s:%s", printValue(v), printType(tp))
}

func printType(tp Type) string {
	switch tp.Kind {
	case TypeI8:
		return "i8"
	case TypeI16:
		return "i16"
	case TypeI32:
		return "i32"
	case TypeI64:
		return "i64"
	case TypeU8:
		return "u8"
	case TypeU16:
		return "u16"
	case TypeU32:
		return "u32"
	case TypeU64:
		return "u64"
	case TypePtr:
		if tp.Elem == nil {
			return "ptr<?>"
		}
		return fmt.Sprintf("ptr<%s>", printType(*tp.Elem))
	default:
		return "<invalid-type>"
	}
}

func opToString(op OpCode) string {
	switch op {
	case OpConst:
		return "const"
	case OpLoad:
		return "load"
	case OpStore:
		return "store"
	case OpAdd:
		return "add"
	case OpSub:
		return "sub"
	case OpMul:
		return "mul"
	case OpDiv:
		return "div"
	case OpNeg:
		return "neg"
	default:
		return fmt.Sprintf("<unknown op %d>", op)
	}
}
