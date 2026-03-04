package lexer

import (
	"strings"
	"testing"

	"github.com/MBlore/AuAu/tokens"
)

func TestLexPackageName(t *testing.T) {
	input := `package "main"`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}

	if result.Tokens[0].Type != tokens.Package {
		t.Errorf("Expected token type %s, got %s", tokens.Package, result.Tokens[0].Type)
	}
	if result.Tokens[1].Type != tokens.String {
		t.Errorf("Expected token type %s, got %s", tokens.String, result.Tokens[1].Type)
	}
}

func TestLexComments(t *testing.T) {
	input := `// Test
	/* Test */
	/* Test */package "main"/* Test */`

	lexer := NewLexer(input)
	result := lexer.Lex()
	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}
}

func TestRowLineReporting(t *testing.T) {
	input := `// New Line
	package "main"`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}

	if result.Tokens[0].Line != 1 || result.Tokens[0].Col != 1 {
		t.Errorf("Expected token to be at line 1, col 1, got line %d, col %d", result.Tokens[0].Line, result.Tokens[0].Col)
	}
	if result.Tokens[1].Line != 1 || result.Tokens[1].Col != 9 {
		t.Errorf("Expected token to be at line 1, col 9, got line %d, col %d", result.Tokens[1].Line, result.Tokens[1].Col)
	}
}

func TestStringTokensHaveBytes(t *testing.T) {
	input := `package "main"`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(result.Tokens))
	}

	if result.Tokens[1].Type != tokens.String || result.Tokens[1].Bytes == nil {
		t.Errorf("Expected string token to have bytes.")
	}
}

func TestStringParseErrorWithNewLines(t *testing.T) {
	input := `package "ma
	in"`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if !strings.Contains(result.Errors[0].Error(), "newline not allowed") {
		t.Errorf("Expected new not allowed error, got '%s'.", result.Errors[0].Error())
	}
}

func TestUnterminatedString(t *testing.T) {
	input := `package "main`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if !strings.Contains(result.Errors[0].Error(), "unterminated string literal") {
		t.Errorf("Expected new not allowed error, got '%s'.", result.Errors[0].Error())
	}
}

func TestIllegalChar(t *testing.T) {
	input := `package "main"¬`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if !strings.Contains(result.Errors[0].Error(), "illegal character") {
		t.Errorf("Expected new not allowed error, got '%s'.", result.Errors[0].Error())
	}
}

func TestUnterminatedCommentBlock(t *testing.T) {
	input := `package "main" /* Comment`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d.", len(result.Errors))
	}
}

func TestLexIdent(t *testing.T) {
	input := `_ident1 ident2`

	lexer := NewLexer(input)
	result := lexer.Lex()

	if len(result.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d.", len(result.Errors))
	}

	if result.Tokens[0].Type != tokens.Ident {
		t.Errorf("Expected ident token, got '%s'.", string(result.Tokens[0].Type))
	}
}

func TestPeekAheadCanReturnZero(t *testing.T) {
	input := `testing`

	lexer := NewLexer(input)

	result := lexer.peekAhead(100)

	if result != 0 {
		t.Errorf("Expected 0, got %d.", result)
	}
}
