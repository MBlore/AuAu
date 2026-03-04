package lexer_test

import (
	"testing"

	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/tokens"
)

func TestLexPackageName(t *testing.T) {
	input := "package main"

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}

	if result.Tokens[0].Type != tokens.Package {
		t.Errorf("Expected token type %s, got %s", tokens.Package, result.Tokens[0].Type)
	}
	if result.Tokens[1].Type != tokens.Ident {
		t.Errorf("Expected token type %s, got %s", tokens.Ident, result.Tokens[1].Type)
	}
}

func TestLexComments(t *testing.T) {
	input := `// Test
	/* Test */
	/* Test */package main/* Test */`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}
}
