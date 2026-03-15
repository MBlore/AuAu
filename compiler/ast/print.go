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

	for _, s := range p.file.Structs {
		p.printStructDecl(s)
	}

	for _, f := range p.file.Functions {
		p.printFuncDecl(f)
	}

	return p.buff.String()
}

func (p *AstPrinter) printStructDecl(s *StructDecl) {
	fmt.Fprintf(&p.buff, "%sStructDecl %s\n", p.prefix(), s.Name)
	p.indentLevel++
	for _, field := range s.Fields {
		fmt.Fprintf(&p.buff, "%sField %s %s\n", p.prefix(), field.Name, TypeToString(field.Type))
	}
	p.indentLevel--
}

func (p *AstPrinter) printFuncDecl(f *FuncDecl) {
	fmt.Fprintf(&p.buff, "%sFuncDecl %s(", p.prefix(), f.Name)

	for i, param := range f.Params {
		p.buff.WriteString(TypeToString(param.Type) + " " + param.Name)
		if i < len(f.Params)-1 {
			p.buff.WriteString(", ")
		}
	}

	fmt.Fprintf(&p.buff, ") return=%s public=%v\n", TypeToString(f.ReturnType), f.IsPublic)

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
	case *ExternFuncStmt:
		fmt.Fprintf(&p.buff, "%sExternFuncStmt %s(", p.prefix(), s.Name)

		for i, param := range s.Params {
			p.buff.WriteString(TypeToString(param.Type) + " " + param.Name)
			if i < len(s.Params)-1 {
				p.buff.WriteString(", ")
			}
		}

		fmt.Fprintf(&p.buff, ") return=%s\n", TypeToString(s.ReturnType))
	case *AssignStmt:
		fmt.Fprintf(&p.buff, "%sAssignStmt\n", p.prefix())
		fmt.Fprintf(&p.buff, "%sTarget:\n", strings.Repeat("  ", p.indentLevel+1))
		p.printExpr(s.Target, p.indentLevel+2)
		fmt.Fprintf(&p.buff, "%sValue:\n", strings.Repeat("  ", p.indentLevel+1))
		p.printExpr(s.Value, p.indentLevel+1)
	case *ForStmt:
		fmt.Fprintf(&p.buff, "%sForStmt\n", p.prefix())
		if s.Init != nil {
			fmt.Fprintf(&p.buff, "%sInit:\n", strings.Repeat("  ", p.indentLevel+1))
			p.indentLevel++
			p.printStmt(s.Init)
			p.indentLevel--
		}
		if s.Cond != nil {
			fmt.Fprintf(&p.buff, "%sCond:\n", strings.Repeat("  ", p.indentLevel+1))
			p.printExpr(s.Cond, p.indentLevel+2)
		}
		if s.Post != nil {
			fmt.Fprintf(&p.buff, "%sPost:\n", strings.Repeat("  ", p.indentLevel+1))
			p.indentLevel++
			p.printStmt(s.Post)
			p.indentLevel--
		}
		fmt.Fprintf(&p.buff, "%sBody:\n", strings.Repeat("  ", p.indentLevel+1))
		p.indentLevel++
		p.printBlock(s.Body)
		p.indentLevel--

	case *WhileStmt:
		fmt.Fprintf(&p.buff, "%sWhileStmt\n", p.prefix())
		fmt.Fprintf(&p.buff, "%sCond:\n", strings.Repeat("  ", p.indentLevel+1))
		p.printExpr(s.Cond, p.indentLevel+2)
		fmt.Fprintf(&p.buff, "%sBody:\n", strings.Repeat("  ", p.indentLevel+1))
		p.indentLevel++
		p.printBlock(s.Body)
		p.indentLevel--

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
			fmt.Fprintf(&p.buff, "%sVarDeclStmt %s %s = \n", p.prefix(), s.Name, TypeToString(s.Type))
			p.printExpr(s.Init, p.indentLevel+1)
		} else {
			fmt.Fprintf(&p.buff, "%sVarDeclStmt %s %s\n", p.prefix(), s.Name, TypeToString(s.Type))
		}
	case *IfStmt:
		fmt.Fprintf(&p.buff, "%sIfStmt\n", p.prefix())
		fmt.Fprintf(&p.buff, "%sCond:\n", strings.Repeat("  ", p.indentLevel+1))
		p.printExpr(s.Cond, p.indentLevel+2)
		fmt.Fprintf(&p.buff, "%sThen:\n", strings.Repeat("  ", p.indentLevel+1))
		p.indentLevel++
		p.printBlock(s.Then)
		p.indentLevel--
		if s.Else != nil {
			fmt.Fprintf(&p.buff, "%sElse:\n", strings.Repeat("  ", p.indentLevel+1))
			p.indentLevel++
			p.printStmt(s.Else)
			p.indentLevel--
		}
	case *BlockStmt:
		p.printBlock(s)
	case *BreakStmt:
		fmt.Fprintf(&p.buff, "%sBreakStmt\n", p.prefix())
	case *ContinueStmt:
		fmt.Fprintf(&p.buff, "%sContinueStmt\n", p.prefix())
	}
}

func (p *AstPrinter) printExpr(expr Expr, indent int) {
	pad := strings.Repeat("  ", indent)

	switch e := expr.(type) {
	case *CallExpr:
		fmt.Fprintf(&p.buff, "%sCallExpr %s\n", pad, e.FuncName)
		for _, arg := range e.Args {
			p.printExpr(arg, indent+1)
		}
	case *IdentExpr:
		fmt.Fprintf(&p.buff, "%sIdentExpr(%s)\n", pad, e.Name)
	case *FieldAccessExpr:
		fmt.Fprintf(&p.buff, "%sFieldAccessExpr(.%s)\n", pad, e.Field)
		p.printExpr(e.Base, indent+1)
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
		fmt.Fprintf(&p.buff, "%sStringLiteralExpr(%q)\n", pad, e.Value)
	default:
		fmt.Fprintf(&p.buff, "%sUnknownExpr(%T)\n", pad, e)
	}
}

func (p *AstPrinter) prefix() string {
	prefix := fmt.Sprintf("%s", string(bytes.Repeat([]byte("  "), p.indentLevel)))
	return prefix
}
