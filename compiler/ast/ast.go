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
	case TypeInt64:
		return "int64"
	case TypeVoid:
		return "void"
	case TypeNull:
		return "null"
	case TypeInt:
		return "int"
	case TypeInt32:
		return "int32"
	case TypeInt16:
		return "int16"
	case TypeInt8:
		return "int8"
	case TypeUInt64:
		return "uint64"
	case TypeUInt32:
		return "uint32"
	case TypeUInt16:
		return "uint16"
	case TypeUInt8:
		return "uint8"
	case TypeBool:
		return "bool"
	case TypeByte:
		return "byte"
	case TypeRune:
		return "rune"
	case TypeString:
		return "string"
	}
	return "unknown"
}

func TypeToString(t *TypeRef) string {
	if t == nil {
		return "nil"
	}

	return TypeKindToString(t.Kind)
}
