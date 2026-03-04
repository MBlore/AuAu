package lowering

import (
	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/ir"
)

// LowerToIR lowers an AST graph to IR instructions.
func LowerToIR(p *ast.SourceFile) (*ir.IRProgram, error) {
	out := &ir.IRProgram{}
	return out, nil
}
