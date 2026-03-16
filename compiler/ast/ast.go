package ast

import (
	"fmt"

	"github.com/MBlore/AuAu/token"
)

// This package contains the AST model of the language.

type TypeKind int

const (
	TypeInvalid TypeKind = iota
	TypeVoid
	TypeNull
	TypeCustom // Used for custom struct types before we resolve them to actual struct definitions.
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
	TypeFloat32
	TypeFloat64
	TypeFloat
)

type TypeRef struct {
	Kind TypeKind
	Name string // For user-defined types, this will be the type name.
}

var (
	TypeVoidRef    = &TypeRef{Kind: TypeVoid}
	TypeIntRef     = &TypeRef{Kind: TypeInt}
	TypeInt64Ref   = &TypeRef{Kind: TypeInt64}
	TypeInt32Ref   = &TypeRef{Kind: TypeInt32}
	TypeInt16Ref   = &TypeRef{Kind: TypeInt16}
	TypeInt8Ref    = &TypeRef{Kind: TypeInt8}
	TypeUInt64Ref  = &TypeRef{Kind: TypeUInt64}
	TypeUInt32Ref  = &TypeRef{Kind: TypeUInt32}
	TypeUInt16Ref  = &TypeRef{Kind: TypeUInt16}
	TypeUInt8Ref   = &TypeRef{Kind: TypeUInt8}
	TypeBoolRef    = &TypeRef{Kind: TypeBool}
	TypeByteRef    = &TypeRef{Kind: TypeByte}
	TypeRuneRef    = &TypeRef{Kind: TypeRune}
	TypeStringRef  = &TypeRef{Kind: TypeString}
	TypeNullRef    = &TypeRef{Kind: TypeNull}
	TypeFloat32Ref = &TypeRef{Kind: TypeFloat32}
	TypeFloat64Ref = &TypeRef{Kind: TypeFloat64}
	TypeFloatRef   = &TypeRef{Kind: TypeFloat}
)

// File is a collection of parsed source code for a single source file.
type File struct {
	PackageName string

	Functions []*FuncDecl
	Externs   []*ExternFuncStmt
	Structs   []*StructDecl
}

func (f *File) Merge(other *File) error {
	// If the package names don't match, we can't merge.
	if f.PackageName != other.PackageName {
		return fmt.Errorf("package name	must be: %s", f.PackageName)
	}

	f.Functions = append(f.Functions, other.Functions...)
	f.Externs = append(f.Externs, other.Externs...)
	f.Structs = append(f.Structs, other.Structs...)

	return nil
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

type FloatLiteralExpr struct {
	NodeMeta
	Literal      string
	InferredType *TypeRef // This will be filled in during type inference.
}

func (*FloatLiteralExpr) isExpr() {}

type StringLiteralExpr struct {
	NodeMeta
	Value string
}

func (*StringLiteralExpr) isExpr() {}

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

type IfStmt struct {
	NodeMeta
	Cond Expr
	Then *BlockStmt
	Else Stmt // nil, *BlockStmt, or *IfStmt
}

func (*IfStmt) isStmt() {}

type WhileStmt struct {
	NodeMeta
	Cond Expr
	Body *BlockStmt
}

func (*WhileStmt) isStmt() {}

type ForStmt struct {
	NodeMeta
	Init Stmt
	Cond Expr
	Post Stmt
	Body *BlockStmt
}

func (*ForStmt) isStmt() {}

type BreakStmt struct {
	NodeMeta
}

func (*BreakStmt) isStmt() {}

type ContinueStmt struct {
	NodeMeta
}

func (*ContinueStmt) isStmt() {}

type AssignStmt struct {
	NodeMeta
	Name   string
	Target Expr
	Value  Expr
}

func (*AssignStmt) isStmt() {}

type ExternFuncStmt struct {
	NodeMeta
	Name       string
	Params     []Param
	ReturnType *TypeRef
}

func (*ExternFuncStmt) isStmt() {}

type CallExpr struct {
	NodeMeta
	FuncName     string
	Args         []Expr
	InferredType *TypeRef
}

func (*CallExpr) isExpr() {}

type StructField struct {
	Name string
	Type *TypeRef
}

type StructDecl struct {
	NodeMeta
	Name   string
	Fields []StructField
}

func (*StructDecl) isStmt() {}

type FieldAccessExpr struct {
	NodeMeta

	// Base is the expression representing the struct instance. For example, in `a.b.c`, `a.b` is the base for the field `c`.
	Base Expr

	Field string
}

func (*FieldAccessExpr) isExpr() {}
