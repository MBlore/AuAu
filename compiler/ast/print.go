package ast

import (
	"bytes"
	"fmt"
	"strings"
)

type AstPrinter struct {
	file        *File
	line        int
	indentLevel int
	buff        strings.Builder
}

func NewAstPrinter(file *File) *AstPrinter {
	return &AstPrinter{
		file: file,
		line: 0,
	}
}

func (p *AstPrinter) Print() string {
	p.buff.WriteString(p.prefix() + "package " + p.file.PackageName + "\n")
	p.line++

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
	p.line++

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
			fmt.Fprintf(&p.buff, "%svar %s %s =", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
			p.printExpr(s.Init)
			p.buff.WriteString("\n")
		} else {
			fmt.Fprintf(&p.buff, "%svar %s %s\n", p.prefix(), s.Name, TypeKindToString(s.Type.Kind))
		}

		p.line++
	}
}

func (p *AstPrinter) printExpr(expr Expr) {
	switch e := expr.(type) {
	case *IdentExpr:
		p.buff.WriteString(" " + e.Name)
	case *IntLiteralExpr:
		p.buff.WriteString(" " + fmt.Sprintf("%d", e.Value))
	case *BinaryExpr:
		p.buff.WriteString(" (")
		p.printExpr(e.Left)
		p.buff.WriteString(" " + string(e.Op) + " ")
		p.printExpr(e.Right)
		p.buff.WriteString(")")
	}
}

func (p *AstPrinter) prefix() string {
	prefix := fmt.Sprintf("%04d: %s", p.line, string(bytes.Repeat([]byte("  "), p.indentLevel)))
	return prefix
}
