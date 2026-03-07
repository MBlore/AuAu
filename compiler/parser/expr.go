package parser

import (
	"errors"
	"strconv"
	"strings"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

// parseExpr will parse all types of RHS expressions, using a Pratt Parsing .
func (p *Parser) parseExpr() (ast.Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == token.Plus {
		op := p.advance()

		right, err := p.parsePrimary()
		if err != nil {
			return nil, errors.New("expected expression after operator")
		}

		left = &ast.BinaryExpr{
			Left:     left,
			Op:       op.Type,
			Right:    right,
			NodeMeta: ast.NodeMeta{Line: op.Line, Col: op.Col},
		}
	}

	return left, nil
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	tok := p.peek()

	switch tok.Type {
	case token.Ident:
		p.advance()
		return &ast.IdentExpr{Name: tok.Literal, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	case token.LParen:
		p.advance()

		// Going in to a new expression inside brackets, so we need to parse it recursively.
		expr, err := p.parseExpr()
		if err != nil {
			return nil, errors.New("expected expression after '('")
		}

		_, err = p.expect(token.RParen)
		if err != nil {
			return nil, errors.New("expected ')' after expression")
		}
		return expr, nil
	case token.Number:
		p.advance()

		lit := tok.Literal
		base := 10

		// Handle 0x, 0c and 0b prefixes for hex, octal and binary literals.
		if strings.HasPrefix(lit, "0x") || strings.HasPrefix(lit, "0X") {
			base = 16
			lit = lit[2:]
		} else if strings.HasPrefix(lit, "0c") || strings.HasPrefix(lit, "0C") {
			base = 8
			lit = lit[2:]
		} else if strings.HasPrefix(lit, "0b") || strings.HasPrefix(lit, "0B") {
			base = 2
			lit = lit[2:]
		}

		val, err := strconv.ParseInt(lit, base, 64)
		if err != nil {
			return nil, errors.New("invalid integer literal")
		}

		return &ast.IntLiteralExpr{Value: val, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	default:
		return nil, errors.New("unexpected token, expected primary expression")
	}
}
