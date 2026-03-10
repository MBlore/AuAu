package ast

import (
	"bytes"
	"fmt"
	"strings"
)

type AstPrinter struct {
	file        *File
	indentLevel int
	buff        strings.Builder
}

func NewAstPrinter(file *File) *AstPrinter {
	return &AstPrinter{
		file: file,
	}
}

func (p *AstPrinter) Print() string {
	p.buff.WriteString(p.prefix() + "package " + p.file.PackageName + "\n")

	for _, f := range p.file.Functions {
		p.printFuncDecl(f)
	}

	return p.buff.String()
}

func (p *AstPrinter) printFuncDecl(f *FuncDecl) {
	fmt.Fprintf(&p.buff, "%sFuncDecl %s(", p.prefix(), f.Name)

	for i, param := range f.Params {
		p.buff.WriteString(TypeKindToString(param.Type.Kind) + " " + param.Name)
		if i < len(f.Params)-1 {
			p.buff.WriteString(", ")
		}
	}

	fmt.Fprintf(&p.buff, ") return=%s public=%v\n", TypeKindToString(f.ReturnType.Kind), f.IsPublic)

	p.printBlock(f.Body)
}

func (p *AstPrinter) printBlock(block *BlockStmt) {
	fmt.Fprintf(&p.buff, "%sBlockStmt\n", p.prefix())
	p.indentLevel++
	for _, stmt := range block.Stmts {
		p.printStmt(stmt)
	}
	p.indentLevel--
}

func (p *AstPrinter) printStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *CallStmt:
		fmt.Fprintf(&p.buff, "%sCallStmt %s\n", p.prefix(), s.FuncName)
		for _, arg := range s.Args {
			p.printExpr(arg, p.indentLevel+1)
		}
	case *ReturnStmt:
		if s.ReturnExpr != nil {
			fmt.Fprintf(&p.buff, "%sReturnStmt\n", p.prefix())
			p.printExpr(s.ReturnExpr, p.indentLevel+1)
		} else {
			fmt.Fprintf(&p.buff, "%sReturnStmt\n", p.prefix())
		}
	case *VarDeclStmt:
		if s.Init != nil {
			fmt.Fprintf(&p.buff, "%sVarDeclStmt %s %s = \n", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
			p.printExpr(s.Init, p.indentLevel+1)
		} else {
			fmt.Fprintf(&p.buff, "%sVarDeclStmt %s %s\n", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
		}
	}
}

func (p *AstPrinter) printExpr(expr Expr, indent int) {
	pad := strings.Repeat("  ", indent)

	switch e := expr.(type) {
	case *IdentExpr:
		fmt.Fprintf(&p.buff, "%sIdentExpr(%s)\n", pad, e.Name)
	case *IntLiteralExpr:
		fmt.Fprintf(&p.buff, "%sIntLiteralExpr(%s)\n", pad, e.Literal)
	case *UnaryExpr:
		fmt.Fprintf(&p.buff, "%sUnaryExpr(%s)\n", pad, TokenTypeToString(e.Op))
		p.printExpr(e.Expr, indent+1)
	case *BinaryExpr:
		fmt.Fprintf(&p.buff, "%sBinaryExpr(%s)\n", pad, TokenTypeToString(e.Op))

		p.printExpr(e.Left, indent+1)
		p.printExpr(e.Right, indent+1)
	case *BoolLiteralExpr:
		fmt.Fprintf(&p.buff, "%sBoolLiteralExpr(%t)\n", pad, e.Value)
	case *StringLiteralExpr:
		fmt.Fprintf(&p.buff, "%sStringLiteralExpr(%s)\n", pad, e.Value)
	default:
		fmt.Fprintf(&p.buff, "%sUnknownExpr(%T)\n", pad, e)
	}
}

func (p *AstPrinter) prefix() string {
	prefix := fmt.Sprintf("%s", string(bytes.Repeat([]byte("  "), p.indentLevel)))
	return prefix
}
