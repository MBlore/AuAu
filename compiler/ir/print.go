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

	case OpConst:
		return fmt.Sprintf("%s = const %d",
			printValue(instr.Dest),
			instr.Const,
		)
	case OpLoad:
		return fmt.Sprintf("%s = load %s",
			printValue(instr.Dest),
			printValue(instr.Args[0]),
		)
	case OpStore:
		return fmt.Sprintf("store %s, %s",
			printValue(instr.Args[0]),
			printValue(instr.Args[1]),
		)
	case OpAdd, OpSub, OpMul, OpDiv:
		return fmt.Sprintf("%s = %s %s, %s",
			printValue(instr.Dest),
			opToString(instr.Op),
			printValue(instr.Args[0]),
			printValue(instr.Args[1]),
		)
	case OpNeg:
		return fmt.Sprintf("%s = neg %s",
			printValue(instr.Dest),
			printValue(instr.Args[0]),
		)
	case OpAlloc:
		return fmt.Sprintf("%s = alloc",
			printValue(instr.Dest),
		)
	default:
		return fmt.Sprintf("%s = <unknown op %d>",
			printValue(instr.Dest),
			instr.Op,
		)
	}
}

func printValue(v IRValue) string {
	return fmt.Sprintf("t%d", v)
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
