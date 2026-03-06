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

	p.buff.WriteString(")\n")
	p.line++
}

func (p *AstPrinter) prefix() string {
	prefix := fmt.Sprintf("%04d: %s", p.line, string(bytes.Repeat([]byte("  "), p.indentLevel)))
	return prefix
}
