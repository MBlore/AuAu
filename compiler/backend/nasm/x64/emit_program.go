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

// asmBlockLabel generates a unique label for a given IR block within a function for use in assembly code.
func asmBlockLabel(fn *ir.Function, block *ir.Block) string {
	if block.Name != "" {
		return fmt.Sprintf("%s_block%d_%s", fn.Name, block.ID, block.Name)
	}
	return fmt.Sprintf("%s_block%d", fn.Name, block.ID)
}
