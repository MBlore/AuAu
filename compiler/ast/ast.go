package ast

// This package contains the AST model of the language.

// SourceFile is a collection of parsed source code for a single source file.
type SourceFile struct {
	PackageName string

	Functions []FuncDecl
}

type Comment struct {
	Text    string
	Line    int
	Col     int
	IsBlock bool
}

// NodeMeta contains the base properties that will be on most AST nodes.
type NodeMeta struct {
	Line            int
	Col             int
	LeadingComments []Comment
}

type FuncDecl struct {
	NodeMeta
	Name string
	// Capitalized function names are public in a package.
	IsPublic bool
}
