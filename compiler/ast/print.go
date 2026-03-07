package ast

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/MBlore/AuAu/token"
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
	fmt.Fprintf(&p.buff, "%sfunc %s(", p.prefix(), f.Name)

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
	p.indentLevel++
	for _, stmt := range block.Stmts {
		p.printStmt(stmt)
	}
	p.indentLevel--
}

func (p *AstPrinter) printStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *VarDeclStmt:
		if s.Init != nil {
			fmt.Fprintf(&p.buff, "%svar %s %s = \n", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
			p.printExpr(s.Init, p.indentLevel+1)
			p.buff.WriteString("\n")
		} else {
			fmt.Fprintf(&p.buff, "%svar %s %s\n", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
		}
	}
}

func (p *AstPrinter) printExpr(expr Expr, indent int) {
	pad := strings.Repeat("  ", indent)

	switch e := expr.(type) {
	case *IdentExpr:
		p.buff.WriteString(fmt.Sprintf("%sIdent(%s)\n", pad, e.Name))
	case *IntLiteralExpr:
		p.buff.WriteString(fmt.Sprintf("%sInt(%d)\n", pad, e.Value))
	case *UnaryExpr:
		p.buff.WriteString(fmt.Sprintf("%sUnary(%s)\n", pad, token.TokenTypeToString(e.Op)))
		p.printExpr(e.Expr, indent+1)
	case *BinaryExpr:
		p.buff.WriteString(fmt.Sprintf("%sBinary(%s)\n", pad, token.TokenTypeToString(e.Op)))

		p.printExpr(e.Left, indent+1)
		p.printExpr(e.Right, indent+1)
	}
}

func (p *AstPrinter) prefix() string {
	prefix := fmt.Sprintf("%s", string(bytes.Repeat([]byte("  "), p.indentLevel)))
	return prefix
}
