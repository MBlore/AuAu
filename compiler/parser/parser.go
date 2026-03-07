package parser

// This package is responsible for taking the lexed tokens and converting them to a valid AST model.

import (
	"errors"
	"fmt"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

type ParseResult struct {
	Errors []error
	File   *ast.File
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

	funcs := []*ast.FuncDecl{}

	for p.peek().Type != token.EOF {
		tok := p.peek()

		// We're only expecting function declarations at the moment.
		f, err := p.parseFuncDecl()
		if err != nil {
			p.addError(tok, err)
			return ParseResult{Errors: p.errors}
		}

		funcs = append(funcs, f)
	}

	sourceFile := ast.File{
		PackageName: tokPackageName.Literal,
		Functions:   funcs,
	}

	return ParseResult{File: &sourceFile, Errors: p.errors}
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

func (p *Parser) parseFuncDecl() (*ast.FuncDecl, error) {
	retType, err := p.parseType()
	if err != nil {
		return nil, errors.New("expected return type for function declaration")
	}

	funcName, err := p.expect(token.Ident)
	if err != nil {
		return nil, errors.New("expected function name after return type")
	}

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}

	_, err = p.expect(token.LBrace)
	if err != nil {
		return nil, errors.New("expected '{' to start function body")
	}

	// Parse statements.
	stmts := []ast.Stmt{}

	for p.peek().Type != token.RBrace {
		st, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		stmts = append(stmts, st)
	}

	_, err = p.expect(token.RBrace)
	if err != nil {
		return nil, errors.New("expected '}' to end function body")
	}

	return &ast.FuncDecl{
		Name:       funcName.Literal,
		ReturnType: retType,
		Params:     params,
		Body:       &ast.BlockStmt{Stmts: stmts},
		IsPublic:   funcName.Literal[0] >= 'A' && funcName.Literal[0] <= 'Z',
	}, nil
}

func (p *Parser) parseStatement() (ast.Stmt, error) {
	tok := p.peek()
	switch tok.Type {
	case token.IntKw:
		p.advance()
		// Expect an identifier for the variable name.
		varName, err := p.expect(token.Ident)
		if err != nil {
			return nil, errors.New("expected variable name after type in variable declaration")
		}

		// Parse the expression initializer if theres an equals sign after the variable name.
		var initExpr ast.Expr
		if p.peek().Type == token.Equals {
			p.advance()

			var err error
			initExpr, err = p.parseExpr(0)
			if err != nil {
				return nil, errors.New("expected initializer expression after '=' in variable declaration")
			}
		}

		return &ast.VarDeclStmt{
			Name: varName.Literal,
			Type: ast.TypeIntRef,
			Init: initExpr,
		}, nil
	default:
		return nil, errors.New("unexpected token, expected statement")
	}
}

func (p *Parser) parseParamList() ([]ast.Param, error) {
	// Expect an opening parenthesis for the parameter list.
	_, err := p.expect(token.LParen)
	if err != nil {
		return nil, errors.New("expected '(' to start parameter list")
	}

	params := []ast.Param{}

	// Parse parameters until we reach the closing parenthesis.
	for p.peek().Type != token.RParen {
		paramType, err := p.parseType()
		if err != nil {
			return nil, errors.New("expected type for parameter")
		}

		paramName, err := p.expect(token.Ident)
		if err != nil {
			return nil, errors.New("expected parameter name after type")
		}

		params = append(params, ast.Param{
			Name: paramName.Literal,
			Type: paramType,
		})

		// Next token must be comma or closing parenthesis.
		if p.peek().Type != token.Comma && p.peek().Type != token.RParen {
			return nil, errors.New("expected ',' or ')' after parameter")
		}

		p.advance()
	}

	// Skip the closing parenthesis.
	p.advance()

	return params, nil
}

// parseType parses a type from the current token position. It returns an error if the token is not a valid type.
func (p *Parser) parseType() (*ast.TypeRef, error) {
	tok := p.peek()
	switch tok.Type {
	case token.Void:
		p.advance()
		return ast.TypeVoidRef, nil
	case token.IntKw:
		p.advance()
		return ast.TypeIntRef, nil
	default:
		return nil, errors.New("unexpected token, expecting type")
	}
}
