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
	externs := []*ast.ExternFuncStmt{}

	for p.peek().Type != token.EOF {
		if p.peek().Type == token.Extern {
			// Handle extern function declarations.
			externFunc, err := p.parseExternFuncDecl()
			if err != nil {
				// Peek on error as parsing would have advanced.
				p.addError(p.peek(), err)
				return ParseResult{Errors: p.errors}
			}

			externs = append(externs, externFunc)
			continue
		}

		// We're only expecting function declarations at the moment.
		f, err := p.parseFuncDecl()
		if err != nil {
			// Peek on error as parsing would have advanced.
			p.addError(p.peek(), err)
			return ParseResult{Errors: p.errors}
		}

		funcs = append(funcs, f)
	}

	sourceFile := ast.File{
		PackageName: tokPackageName.Literal,
		Functions:   funcs,
		Externs:     externs,
	}

	return ParseResult{File: &sourceFile, Errors: p.errors}
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

	blk, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.FuncDecl{
		Name:       funcName.Literal,
		ReturnType: retType,
		Params:     params,
		Body:       blk,
		IsPublic:   funcName.Literal[0] >= 'A' && funcName.Literal[0] <= 'Z',
	}, nil
}

func (p *Parser) parseBlock() (*ast.BlockStmt, error) {
	// Expect an opening curly brace for the block.
	_, err := p.expect(token.LBrace)
	if err != nil {
		return nil, errors.New("expected '{', got '" + p.peek().Literal + "'")
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
		return nil, errors.New("expected '}' to end function body, got '" + p.peek().Literal + "'")
	}

	return &ast.BlockStmt{Stmts: stmts}, nil
}

func (p *Parser) parseStatement() (ast.Stmt, error) {
	tok := p.peek()
	switch tok.Type {
	case token.Break:
		p.advance()
		return &ast.BreakStmt{}, nil

	case token.Continue:
		p.advance()
		return &ast.ContinueStmt{}, nil

	case token.While:
		return p.parseWhileStmt()

	case token.If:
		return p.parseIfStmt()

	case token.For:
		return p.parseForStmt()

	case token.Ident:
		// Assign?
		if p.peekAhead(1).Type == token.Equals {
			varName := tok.Literal

			p.advance()
			p.advance()

			initExpr, err := p.parseNewExpr()
			if err != nil {
				return nil, fmt.Errorf("invalid initializer expression in assignment statement: %w", err)
			}

			return &ast.AssignStmt{
				Name:  varName,
				Value: initExpr,
			}, nil
		}

		// Function call?
		if p.peekAhead(1).Type == token.LParen {
			expr, err := p.parseNewExpr()
			if err != nil {
				return nil, fmt.Errorf("invalid function call expression: %w", err)
			}

			call, ok := expr.(*ast.CallExpr)
			if !ok {
				return nil, errors.New("expected function call")
			}

			return &ast.CallStmt{
				FuncName: call.FuncName,
				Args:     call.Args,
				NodeMeta: call.NodeMeta,
			}, nil
		}

		return nil, errors.New("unexpected identifier")
	case token.Return:

		p.advance()

		var returnExpr ast.Expr

		if p.isExprStart(p.peek().Type) {
			expr, err := p.parseNewExpr()
			if err != nil {
				return nil, fmt.Errorf("invalid return expression: %w", err)
			}

			// Sometimes people will think the language supports multiple return with this kind of syntax, so
			// we can be helpful and let them know we don't have that feature if they try to use it.
			if p.peek().Type == token.Comma {
				return nil, errors.New("unexpected ',' after return expression, multiple return types are not allowed")
			}

			returnExpr = expr
		}

		return &ast.ReturnStmt{ReturnExpr: returnExpr}, nil

	case token.Extern:
		return nil, errors.New("unexpected 'extern' keyword, extern function declarations must be at the file level")

	default:
		// Assume its a variable definition for now.
		varType, err := p.parseType()
		if err != nil {
			return nil, errors.New("expected type, found unexpected token: " + tok.Literal)
		}

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
			initExpr, err = p.parseNewExpr()
			if err != nil {
				return nil, errors.New("invalid initializer expression after '=' in variable declaration")
			}
		}

		return &ast.VarDeclStmt{
			Name: varName.Literal,
			Type: varType,
			Init: initExpr,
		}, nil
	}
}

func (p *Parser) parseExternFuncDecl() (*ast.ExternFuncStmt, error) {
	p.advance()

	retType, err := p.parseType()
	if err != nil {
		return nil, errors.New("expected return type for extern function declaration")
	}

	funcName, err := p.expect(token.Ident)
	if err != nil {
		return nil, errors.New("expected function name after return type in extern function declaration")
	}

	params, err := p.parseParamList()
	if err != nil {
		return nil, err
	}

	return &ast.ExternFuncStmt{
		Name:       funcName.Literal,
		ReturnType: retType,
		Params:     params,
	}, nil
}

func (p *Parser) parseForStmt() (ast.Stmt, error) {
	// Format is: for int i = 0; i < 5; i = i + 1 { ... }
	p.advance()
	init, err := p.parseStatement()
	if err != nil {
		return nil, fmt.Errorf("invalid initializer statement in for loop: %w", err)
	}

	p.expect(token.Semicolon)

	cond, err := p.parseNewExpr()
	if err != nil {
		return nil, fmt.Errorf("invalid condition expression in for loop: %w", err)
	}

	p.expect(token.Semicolon)

	post, err := p.parseStatement()
	if err != nil {
		return nil, fmt.Errorf("invalid post statement in for loop: %w", err)
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("invalid block in for loop: %w", err)
	}

	return &ast.ForStmt{
		Init: init,
		Cond: cond,
		Post: post,
		Body: body,
	}, nil
}

func (p *Parser) parseIfStmt() (ast.Stmt, error) {
	p.advance()
	cond, err := p.parseNewExpr()
	if err != nil {
		return nil, fmt.Errorf("invalid condition expression in if statement: %w", err)
	}

	thenBlock, err := p.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("invalid block in if statement: %w", err)
	}

	if p.peek().Type == token.Else {
		p.advance()

		if p.peek().Type == token.If {
			// Handle else if by recursively parsing another if statement as the else block.
			elseIfStmt, err := p.parseIfStmt()
			if err != nil {
				return nil, fmt.Errorf("invalid else if statement: %w", err)
			}

			return &ast.IfStmt{
				Cond: cond,
				Then: thenBlock,
				Else: elseIfStmt,
			}, nil
		} else {
			// Single else block.
			elseBlock, err := p.parseBlock()
			if err != nil {
				return nil, fmt.Errorf("invalid else block: %w", err)
			}

			return &ast.IfStmt{
				Cond: cond,
				Then: thenBlock,
				Else: elseBlock,
			}, nil
		}
	} else {
		return &ast.IfStmt{
			Cond: cond,
			Then: thenBlock,
		}, nil
	}
}

func (p *Parser) parseWhileStmt() (ast.Stmt, error) {
	p.advance()

	cond, err := p.parseNewExpr()
	if err != nil {
		return nil, fmt.Errorf("invalid condition expression in while statement: %w", err)
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, fmt.Errorf("invalid block in while statement: %w", err)
	}

	return &ast.WhileStmt{
		Cond: cond,
		Body: body,
	}, nil
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
		next := p.peek().Type
		if next != token.Comma && next != token.RParen {
			return nil, errors.New("expected ',' or ')' after parameter")
		}

		if next == token.Comma {
			p.advance()
		}

		if next == token.RParen {
			break
		}
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
	case token.StringKw:
		p.advance()
		return ast.TypeStringRef, nil
	case token.BoolKw:
		p.advance()
		return ast.TypeBoolRef, nil
	case token.ByteKw:
		p.advance()
		return ast.TypeByteRef, nil
	case token.RuneKw:
		p.advance()
		return ast.TypeRuneRef, nil
	case token.UInt8Kw:
		p.advance()
		return ast.TypeUInt8Ref, nil
	case token.UInt16Kw:
		p.advance()
		return ast.TypeUInt16Ref, nil
	case token.UInt32Kw:
		p.advance()
		return ast.TypeUInt32Ref, nil
	case token.UInt64Kw:
		p.advance()
		return ast.TypeUInt64Ref, nil
	case token.Int8Kw:
		p.advance()
		return ast.TypeInt8Ref, nil
	case token.Int16Kw:
		p.advance()
		return ast.TypeInt16Ref, nil
	case token.Int32Kw:
		p.advance()
		return ast.TypeInt32Ref, nil
	case token.Int64Kw:
		p.advance()
		return ast.TypeInt64Ref, nil
	case token.Float32Kw:
		p.advance()
		return ast.TypeFloat32Ref, nil
	case token.Float64Kw:
		p.advance()
		return ast.TypeFloat64Ref, nil
	case token.FloatKw:
		p.advance()
		return ast.TypeFloatRef, nil
	default:
		return nil, errors.New("unexpected token, expecting type")
	}
}
