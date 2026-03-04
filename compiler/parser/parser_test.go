package parser

import (
	"strings"
	"testing"

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

	if pr.SourceFile == nil {
		t.Errorf("Expected a SourceFile, got nil")
	}

	if pr.SourceFile.PackageName != "main" {
		t.Errorf("Expected package name 'main', got '%s'", pr.SourceFile.PackageName)
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
