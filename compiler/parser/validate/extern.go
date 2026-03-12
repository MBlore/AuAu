package validate

// ensureUniqueExternFuncNames checks that all ExternFuncStmt in the file have unique names.
func ensureUniqueExternFuncNames(ctx *validateContext) {
	nameSet := make(map[string]struct{})

	for _, stmt := range ctx.file.Externs {
		if _, exists := nameSet[stmt.Name]; exists {
			ctx.addError(stmt.NodeMeta, "duplicate extern function name: "+stmt.Name)
		} else {
			nameSet[stmt.Name] = struct{}{}
		}
	}
}
