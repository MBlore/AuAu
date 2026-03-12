package parser

import (
	"errors"
	"fmt"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

// isExprStart checks if the given token type can be the start of an expression.
func (p *Parser) isExprStart(t token.TokenType) bool {
	switch t {
	case token.Ident,
		token.Number,
		token.LParen,
		token.Sub,
		token.Float,
		token.True,
		token.False,
		token.String:
		return true
	default:
		return false
	}
}

// addError adds an error to the parsers error list with file, line, and column information
// from the provided token.
func (p *Parser) addError(tok token.Token, err error) {
	strError := fmt.Sprintf("%s:%d:%d: %s", p.filename, tok.Line, tok.Col, err.Error())
	p.errors = append(p.errors, errors.New(strError))
}

// expect validates the current position token to be the specified token type. If it is, the parsing
// position is advanced to the next token. If not, an error is returned.
func (p *Parser) expect(tt token.TokenType) (token.Token, error) {
	tok := p.peek()
	if tok.Type != tt {
		return tok, errors.New("unexpected token, got '" + tok.Literal + "', expected '" + ast.TokenTypeToString(tt) + "'")
	}

	p.advance()
	return tok, nil
}

// advance moves the token index forward by one.
func (p *Parser) advance() token.Token {
	if p.pos < len(p.tokens) {
		p.pos++
	}

	return p.peek()
}

// peek returns the token at the current parsing position without consuming it.
func (p *Parser) peek() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}

	return p.tokens[p.pos]
}

// peekOffset returns the token at the current parsing position plus the offset, without consuming it.
func (p *Parser) peekAhead(offset int) token.Token {
	if p.pos+offset >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}

	return p.tokens[p.pos+offset]
}
