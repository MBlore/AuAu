package ast

import "github.com/MBlore/AuAu/token"

// This package contains the AST model of the language.

type TypeKind int

const (
	TypeInvalid TypeKind = iota
	TypeVoid
	TypeInt
)

type TypeRef struct {
	Kind TypeKind
}

var (
	TypeVoidRef = &TypeRef{Kind: TypeVoid}
	TypeIntRef  = &TypeRef{Kind: TypeInt}
)

// File is a collection of parsed source code for a single source file.
type File struct {
	PackageName string

	Functions []*FuncDecl
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
	Name       string
	Params     []Param
	ReturnType *TypeRef
	Body       *BlockStmt
	IsPublic   bool // Capitalized function names are public in a package.
}

type Param struct {
	Name string
	Type *TypeRef
}

type Stmt interface {
	isStmt()
}

type BlockStmt struct {
	NodeMeta
	Stmts []Stmt
}

func (*BlockStmt) isStmt() {}

type VarDeclStmt struct {
	NodeMeta
	Name string
	Type *TypeRef
	Init Expr
}

func (*VarDeclStmt) isStmt() {}

type Expr interface {
	isExpr()
}

type IntLiteralExpr struct {
	NodeMeta
	Value int64
}

func (*IntLiteralExpr) isExpr() {}

type IdentExpr struct {
	NodeMeta
	Name string
}

func (*IdentExpr) isExpr() {}

type BinaryExpr struct {
	NodeMeta
	Left  Expr
	Right Expr
	Op    token.TokenType
}

func (*BinaryExpr) isExpr() {}

type UnaryExpr struct {
	NodeMeta
	Expr Expr
	Op   token.TokenType
}

func (*UnaryExpr) isExpr() {}

func TypeKindToString(t TypeKind) string {
	switch t {
	case TypeInt:
		return "int"
	case TypeVoid:
		return "void"
	}
	return "unknown"
}

func TypeToString(t *TypeRef) string {
	if t == nil {
		return "nil"
	}

	return TypeKindToString(t.Kind)
}
