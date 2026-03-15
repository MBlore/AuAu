package validate

import "github.com/MBlore/AuAu/ast"

func ensureUniqueStructNames(ctx *validateContext) {
	structNames := make(map[string]bool)

	for _, decl := range ctx.file.Structs {
		if structNames[decl.Name] {
			ctx.addError(decl.NodeMeta, "duplicate struct name: "+decl.Name)
			continue
		}

		structNames[decl.Name] = true
	}
}

func ensureKnownCustomTypes(ctx *validateContext) {
	for _, decl := range ctx.file.Structs {
		for _, field := range decl.Fields {
			validateKnownType(ctx, field.Type, decl.NodeMeta, "unknown type in struct field "+decl.Name+"."+field.Name+": ")
		}
	}

	for _, fn := range ctx.file.Functions {
		validateKnownType(ctx, fn.ReturnType, fn.NodeMeta, "unknown return type for function "+fn.Name+": ")

		for _, param := range fn.Params {
			validateKnownType(ctx, param.Type, fn.NodeMeta, "unknown parameter type for function "+fn.Name+" parameter "+param.Name+": ")
		}

		ensureKnownCustomTypesInBlock(ctx, fn.Body)
	}

	for _, ext := range ctx.file.Externs {
		validateKnownType(ctx, ext.ReturnType, ext.NodeMeta, "unknown return type for extern "+ext.Name+": ")

		for _, param := range ext.Params {
			validateKnownType(ctx, param.Type, ext.NodeMeta, "unknown parameter type for extern "+ext.Name+" parameter "+param.Name+": ")
		}
	}
}

func ensureKnownCustomTypesInBlock(ctx *validateContext, block *ast.BlockStmt) {
	for _, stmt := range block.Stmts {
		ensureKnownCustomTypesInStmt(ctx, stmt)
	}
}

func ensureKnownCustomTypesInStmt(ctx *validateContext, stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.VarDeclStmt:
		validateKnownType(ctx, s.Type, s.NodeMeta, "unknown variable type for "+s.Name+": ")
	case *ast.BlockStmt:
		ensureKnownCustomTypesInBlock(ctx, s)
	case *ast.IfStmt:
		ensureKnownCustomTypesInBlock(ctx, s.Then)
		if s.Else != nil {
			ensureKnownCustomTypesInStmt(ctx, s.Else)
		}
	case *ast.WhileStmt:
		ensureKnownCustomTypesInBlock(ctx, s.Body)
	case *ast.ForStmt:
		if s.Init != nil {
			ensureKnownCustomTypesInStmt(ctx, s.Init)
		}
		if s.Post != nil {
			ensureKnownCustomTypesInStmt(ctx, s.Post)
		}
		if s.Body != nil {
			ensureKnownCustomTypesInBlock(ctx, s.Body)
		}
	}
}

func validateKnownType(ctx *validateContext, typ *ast.TypeRef, nodeMeta ast.NodeMeta, prefix string) {
	if typ == nil || typ.Kind != ast.TypeCustom {
		return
	}

	if _, ok := ctx.structs[typ.Name]; !ok {
		ctx.addError(nodeMeta, prefix+typ.Name)
	}
}
