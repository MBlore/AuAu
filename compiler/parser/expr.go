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

	// First we parse the left-hand side of the expression,
	// which can be a primary expression (identifier, literal, parenthesized expression)
	// or a prefix expression. A prefix expression is an operator that comes before its operand,
	// like "-a" or "!b".
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// After parsing the left-hand side, we check if there are any postfix operators that can be applied to it,
	// such as function calls or field accesses. We loop to handle multiple postfix operators in a row,
	// e.g. "a.b.c()" would be parsed as a field access of "a" to "b", followed by another field access to "c",
	// followed by a function call.
	left, err = p.parsePostfix(left)
	if err != nil {
		return nil, err
	}

	// Now we check if there are any infix operators that can be applied to the left-hand side.
	// Infix operators are binary operators that come between their operands, like "a + b" or "x * y".
	// We loop to handle multiple binary operators in a row, e.g. "a + b * c - d".
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
func (p *Parser) parsePrimary() (ast.Expr, error) {
	tok := p.peek()

	switch tok.Type {
	case token.Ident:
		p.advance()

		return &ast.IdentExpr{
			Name:     tok.Literal,
			NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col},
		}, nil

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

	case token.Float:
		p.advance()
		return &ast.FloatLiteralExpr{Literal: tok.Literal, InferredType: nil, NodeMeta: ast.NodeMeta{Line: tok.Line, Col: tok.Col}}, nil

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

// parseCallSuffix parses the suffix of a function call after the function name has been parsed as an identifier.
func (p *Parser) parseCallSuffix(callee *ast.IdentExpr) (ast.Expr, error) {
	call := &ast.CallExpr{
		FuncName: callee.Name,
		NodeMeta: callee.NodeMeta,
	}

	p.advance() // skip '('

	args := []ast.Expr{}

	for p.peek().Type != token.RParen {
		arg, err := p.parseNewExpr()
		if err != nil {
			return nil, fmt.Errorf("invalid argument expression in function call: %w", err)
		}

		args = append(args, arg)

		if p.peek().Type != token.Comma && p.peek().Type != token.RParen {
			return nil, errors.New("expected ',' or ')' after function call argument")
		}

		if p.peek().Type == token.Comma {
			p.advance()
		}
	}

	p.advance() // skip ')'

	call.Args = args
	return call, nil
}

// parsePostfix parses postfix expressions like function calls and field accesses after the primary expression has been parsed.
func (p *Parser) parsePostfix(expr ast.Expr) (ast.Expr, error) {
	for {
		switch p.peek().Type {
		case token.LParen:
			ident, ok := expr.(*ast.IdentExpr)
			if !ok {
				return nil, errors.New("expected function name before '('")
			}

			nextExpr, err := p.parseCallSuffix(ident)
			if err != nil {
				return nil, err
			}
			expr = nextExpr

		case token.Dot:
			nextExpr, err := p.parseFieldAccessSuffix(expr)
			if err != nil {
				return nil, err
			}
			expr = nextExpr

		default:
			return expr, nil
		}
	}
}

// parseFieldAccessSuffix parses the suffix of a field access after the base expression has been parsed.
// For example, in "a.b", after parsing "a" as the base expression, this function will parse the ".b" part
// and return a FieldAccessExpr.
func (p *Parser) parseFieldAccessSuffix(base ast.Expr) (ast.Expr, error) {
	dotTok := p.peek()
	p.advance() // skip '.'

	fieldTok, err := p.expect(token.Ident)
	if err != nil {
		return nil, fmt.Errorf("expected field name after '.': %w", err)
	}

	return &ast.FieldAccessExpr{
		Base:  base,
		Field: fieldTok.Literal,
		NodeMeta: ast.NodeMeta{
			Line: dotTok.Line,
			Col:  dotTok.Col,
		},
	}, nil
}
