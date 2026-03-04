package parser

// This package is responsible for taking the lexed tokens and converting them to a valid AST model.

import (
	"errors"
	"fmt"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

type ParseResult struct {
	Errors     []error
	SourceFile *ast.SourceFile
}

type Parser struct {
	// The filename of the source file the tokens were lexed from.
	filename string
	tokens   []token.Token
	pos      int
	errors   []error
}

func NewParser(filename string, tokens []token.Token) *Parser {
	parser := &Parser{filename: filename, tokens: tokens}
	return parser
}

// Parse will parse the tokens and build a valid AST graph.
func (p *Parser) Parse() ParseResult {
	// Always expect a package name as the first token.
	tokPackage, err := p.expect(token.Package)
	if err != nil {
		p.addError(tokPackage, errors.New("package declaration must be the first statement in a source file, e.g. 'package \"main\"'"))
		return ParseResult{Errors: p.errors}
	}

	// ...followed by the package name.
	tokPackageName, err := p.expect(token.String)
	if err != nil {
		p.addError(tokPackageName, errors.New("expected package name string after 'package' keyword, e.g. 'package \"main\"'"))
		return ParseResult{Errors: p.errors}
	}

	// TODO: Parse functions.

	sourceFile := ast.SourceFile{
		PackageName: tokPackageName.Literal,
	}

	return ParseResult{SourceFile: &sourceFile, Errors: p.errors}
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
		return tok, errors.New("unexpected token")
	}

	p.advance()
	return tok, nil
}

// advance moves the token index forward by one.
func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
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
