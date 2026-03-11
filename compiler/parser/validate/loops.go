package validate

import (
	"errors"

	"github.com/MBlore/AuAu/ast"
)

func ensureLoopControlUsedInsideLoop(ctx *validateContext) {
	for _, fn := range ctx.file.Functions {
		checkLoopControlInBlock(ctx, fn.Body, 0)
	}
}

func checkLoopControlInBlock(ctx *validateContext, block *ast.BlockStmt, loopDepth int) {
	for _, stmt := range block.Stmts {
		checkLoopControlInStmt(ctx, stmt, loopDepth)
	}
}

func checkLoopControlInStmt(ctx *validateContext, stmt ast.Stmt, loopDepth int) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		checkLoopControlInBlock(ctx, s, loopDepth)

	case *ast.IfStmt:
		checkLoopControlInBlock(ctx, s.Then, loopDepth)
		if s.Else != nil {
			checkLoopControlInStmt(ctx, s.Else, loopDepth)
		}

	case *ast.WhileStmt:
		checkLoopControlInBlock(ctx, s.Body, loopDepth+1)

	case *ast.ForStmt:
		if s.Init != nil {
			checkLoopControlInStmt(ctx, s.Init, loopDepth+1)
		}
		if s.Body != nil {
			checkLoopControlInBlock(ctx, s.Body, loopDepth+1)
		}
		if s.Post != nil {
			checkLoopControlInStmt(ctx, s.Post, loopDepth+1)
		}

	case *ast.BreakStmt:
		if loopDepth == 0 {
			ctx.errors = append(ctx.errors, errors.New("break used outside of loop"))
		}

	case *ast.ContinueStmt:
		if loopDepth == 0 {
			ctx.errors = append(ctx.errors, errors.New("continue used outside of loop"))
		}
	}
}
