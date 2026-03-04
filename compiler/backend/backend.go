package backend

import (
	"bytes"
	"os"

	"github.com/MBlore/AuAu/ir"
)

// CompileToASM compiles an IR program to an assembly file.
func CompileToASM(outFilename string, program *ir.IRProgram) error {
	var b bytes.Buffer

	b.WriteString("global main\n")
	b.WriteString("section .rdata\n")
	b.WriteString("section .text\n")
	b.WriteString("main:\n")
	b.WriteString("  ret\n")

	err := os.WriteFile(outFilename, b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
