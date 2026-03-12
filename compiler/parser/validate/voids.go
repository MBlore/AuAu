package validate

import "github.com/MBlore/AuAu/ast"

func ensureNoVoidVariables(ctx *validateContext) {
	for _, fn := range ctx.file.Functions {
		checkBlockForVoidVariables(ctx, fn.Body)
	}
}

func checkBlockForVoidVariables(ctx *validateContext, block *ast.BlockStmt) {
	for _, stmt := range block.Stmts {
		checkStmtForVoidVariables(ctx, stmt)
	}
}

func checkStmtForVoidVariables(ctx *validateContext, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		if s.Type.Kind == ast.TypeVoid {
			ctx.addError(s.NodeMeta, "variable cannot be declared with void type: "+s.Name)
		}

	case *ast.BlockStmt:
		checkBlockForVoidVariables(ctx, s)

	case *ast.IfStmt:
		checkBlockForVoidVariables(ctx, s.Then)

		if s.Else != nil {
			checkStmtForVoidVariables(ctx, s.Else)
		}

	case *ast.WhileStmt:
		checkBlockForVoidVariables(ctx, s.Body)

	case *ast.ForStmt:
		if s.Init != nil {
			checkStmtForVoidVariables(ctx, s.Init)
		}
		if s.Body != nil {
			checkBlockForVoidVariables(ctx, s.Body)
		}
		if s.Post != nil {
			checkStmtForVoidVariables(ctx, s.Post)
		}
	}
}
