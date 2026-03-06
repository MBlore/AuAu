package parser

import (
	"strings"
	"testing"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/token"
)

func TestSuccessParse(t *testing.T) {
	code := `package "main"`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if pr.File == nil {
		t.Errorf("Expected a SourceFile, got nil")
	}

	if pr.File.PackageName != "main" {
		t.Errorf("Expected package name 'main', got '%s'", pr.File.PackageName)
	}
}

func TestPackageFirstTokenError(t *testing.T) {
	code := "test"
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 1 || !strings.Contains(pr.Errors[0].Error(), "package declaration must be the first statement") {
		t.Errorf("Expected 1 error about package declaration, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestPackageFormat(t *testing.T) {
	code := "package main"
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 1 || !strings.Contains(pr.Errors[0].Error(), "expected package name string") {
		t.Errorf("Expected 1 error about package declaration, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestPeakAheadReturnsEOF(t *testing.T) {
	code := `package "main"`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)

	if parser.peekAhead(100).Type != token.EOF {
		t.Errorf("Expected peekAhead(100) to return EOF, got %v", parser.peekAhead(100))
	}
}

func TestPeakReturnsEOF(t *testing.T) {
	code := `package "main"`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)

	// Advance to the end of the tokens.
	for i := 0; i < len(lexResult.Tokens); i++ {
		parser.advance()
	}

	if parser.peek().Type != token.EOF {
		t.Errorf("Expected peek() at end of tokens to return EOF, got %v", parser.peek())
	}
}

func TestReadOneFunc(t *testing.T) {
	input := `package "main"

	void main() {
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if pr.File == nil {
		t.Errorf("Expected a File, got nil")
	}

	if len(pr.File.Functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(pr.File.Functions))
	}

	mainFunc := pr.File.Functions[0]
	if mainFunc.Name != "main" {
		t.Errorf("Expected function name 'main', got '%s'", mainFunc.Name)
	}
}

func TestReadTwoFuncs(t *testing.T) {
	input := `package "main"

	void main() {
	}

	int Foo() {
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.File.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(pr.File.Functions))
	}

	mainFunc := pr.File.Functions[0]
	if mainFunc.Name != "main" {
		t.Errorf("Expected first function name 'main', got '%s'", mainFunc.Name)
	}
	if mainFunc.IsPublic != false {
		t.Errorf("Expected 'main' function to be private, got IsPublic=%v", mainFunc.IsPublic)
	}
	if mainFunc.ReturnType.Kind != ast.TypeVoid {
		t.Errorf("Expected 'main' function return type to be 'void', got %v", mainFunc.ReturnType.Kind)
	}

	fooFunc := pr.File.Functions[1]
	if fooFunc.Name != "Foo" {
		t.Errorf("Expected second function name 'Foo', got '%s'", fooFunc.Name)
	}
	if fooFunc.IsPublic != true {
		t.Errorf("Expected 'Foo' function to be public, got IsPublic=%v", fooFunc.IsPublic)
	}
	if fooFunc.ReturnType.Kind != ast.TypeInt {
		t.Errorf("Expected 'Foo' function return type to be 'int', got %v", fooFunc.ReturnType.Kind)
	}
}
