package ast

import "github.com/MBlore/AuAu/token"

// This package contains the AST model of the language.

type TypeKind int

const (
	TypeInvalid TypeKind = iota
	TypeVoid
	TypeNull

	TypeInt
	TypeInt64
	TypeInt32
	TypeInt16
	TypeInt8
	TypeUInt64
	TypeUInt32
	TypeUInt16
	TypeUInt8

	TypeBool
	TypeByte
	TypeRune
	TypeString
)

type TypeRef struct {
	Kind TypeKind
}

var (
	TypeVoidRef   = &TypeRef{Kind: TypeVoid}
	TypeIntRef    = &TypeRef{Kind: TypeInt}
	TypeInt64Ref  = &TypeRef{Kind: TypeInt64}
	TypeInt32Ref  = &TypeRef{Kind: TypeInt32}
	TypeInt16Ref  = &TypeRef{Kind: TypeInt16}
	TypeInt8Ref   = &TypeRef{Kind: TypeInt8}
	TypeUInt64Ref = &TypeRef{Kind: TypeUInt64}
	TypeUInt32Ref = &TypeRef{Kind: TypeUInt32}
	TypeUInt16Ref = &TypeRef{Kind: TypeUInt16}
	TypeUInt8Ref  = &TypeRef{Kind: TypeUInt8}
	TypeBoolRef   = &TypeRef{Kind: TypeBool}
	TypeByteRef   = &TypeRef{Kind: TypeByte}
	TypeRuneRef   = &TypeRef{Kind: TypeRune}
	TypeStringRef = &TypeRef{Kind: TypeString}
	TypeNullRef   = &TypeRef{Kind: TypeNull}
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

type ReturnStmt struct {
	NodeMeta
	ReturnExpr Expr
}

func (*ReturnStmt) isStmt() {}

type CallStmt struct {
	NodeMeta
	FuncName string
	Args     []Expr
}

func (*CallStmt) isStmt() {}

type Expr interface {
	isExpr()
}

type IntLiteralExpr struct {
	NodeMeta
	Literal      string
	Base         int
	InferredType *TypeRef // This will be filled in during type inference.
}

func (*IntLiteralExpr) isExpr() {}

type BoolLiteralExpr struct {
	NodeMeta
	Value bool
}

func (*BoolLiteralExpr) isExpr() {}

type IdentExpr struct {
	NodeMeta
	Name string
}

func (*IdentExpr) isExpr() {}

type BinaryExpr struct {
	NodeMeta
	Left         Expr
	Right        Expr
	Op           token.TokenType
	InferredType *TypeRef
}

func (*BinaryExpr) isExpr() {}

type UnaryExpr struct {
	NodeMeta
	Expr         Expr
	Op           token.TokenType
	InferredType *TypeRef
}

func (*UnaryExpr) isExpr() {}
