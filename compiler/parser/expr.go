package parser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/token"
)

// parseNewExpr is the entry point for parsing an expression.
// It will call parseExpr with a minimum binding power of 0 to start parsing the expression.
func (p *Parser) parseNewExpr() (ast.Expr, error) {
	return p.parseExpr(0)
}

// parseExpr will parse all types of RHS expressions, using a Pratt Parsing method.
// It looks across multiple tokens and builds the binary operations tree.
func (p *Parser) parseExpr(minBP int) (ast.Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		op := p.peek()
		bp := infixBindingPower(op.Type)

		// If the operator's binding power is less than the minimum binding power, we stop parsing.
		if bp < minBP {
			break
		}

		// Consumes the operator.
		p.advance()

		// We add 1 to enforce left-associativity which means that in an expression like "a - b - c",
		// the first "-" operator will bind more tightly to "a" and "b" than the second "-" operator,
		// resulting in the correct grouping of "(a - b) - c".
		right, err := p.parseExpr(bp + 1)
		if err != nil {
			return nil, fmt.Errorf("expected expression after operator: %w", err)
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

// parsePrimary parses primary expressions: identifiers, literals, and parenthesized expressions.
// Note however that paranthesized expressions are parsed recursively, so they can contain any expression, not just primaries.
func (p *Parser) parsePrimary() (ast.Expr, error) {
	tok := p.peek()

	switch tok.Type {
	case token.Ident:
		p.advance()
		return &ast.IdentExpr{Name: tok.Literal, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	case token.LParen:
		p.advance()

		// Going in to a new expression inside brackets, so we need to parse it recursively.
		expr, err := p.parseNewExpr()
		if err != nil {
			return nil, fmt.Errorf("expected expression after '(': %w", err)
		}

		_, err = p.expect(token.RParen)
		if err != nil {
			return nil, fmt.Errorf("expected ')' after expression: %w", err)
		}
		return expr, nil
	case token.Sub:
		p.advance()

		bp := prefixBindingPower(tok.Type)

		right, err := p.parseExpr(bp)
		if err != nil {
			return nil, fmt.Errorf("expected expression after prefix operator: %w", err)
		}

		return &ast.UnaryExpr{
			Expr:     right,
			Op:       tok.Type,
			NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col},
		}, nil

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

		return &ast.IntLiteralExpr{Literal: lit, Base: base, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	case token.True, token.False:
		p.advance()
		return &ast.BoolLiteralExpr{Value: tok.Type == token.True, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	case token.String:
		p.advance()
		return &ast.StringLiteralExpr{Value: tok.Literal, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil
	default:
		return nil, errors.New("unexpected token, expected primary expression")
	}
}

// infixBindingPower returns the binding power of an infix operator for Pratt parsing.
func infixBindingPower(op token.TokenType) int {
	switch op {
	case token.OrOr:
		return 5
	case token.AndAnd:
		return 10
	case token.EqEq, token.NotEq:
		return 15
	case token.Lt, token.LtEq, token.Gt, token.GtEq:
		return 20
	case token.Add, token.Sub:
		return 30
	case token.Mul, token.Div:
		return 40
	}

	// Not an infix operator.
	return -1
}

// prefixBindingPower returns the binding power of a prefix operator for Pratt parsing.
func prefixBindingPower(op token.TokenType) int {
	switch op {
	case token.Sub:
		return 30
	}

	// Not a prefix operator.
	return -1
}
