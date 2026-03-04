package lexer

import (
	"errors"
	"unicode"

	"github.com/MBlore/AuAu/tokens"
)

type Lexer struct {
	src  []rune
	pos  int // current position in input (points to current char)
	col  int
	line int
}

type LexResult struct {
	Tokens []tokens.Token
	Errors []error
}

// NewLexer creates a new Lexer instance with the given input source code.
func NewLexer(src string) *Lexer {
	l := &Lexer{src: []rune(src)}
	return l
}

// Lex takes the input source code and produces a list of tokens.
func (l *Lexer) Lex() LexResult {
	toks := []tokens.Token{}
	errs := []error{}

	for {
		tok, err := l.nextToken()
		if err != nil {
			errs = append(errs, err)
			break
		}

		if tok.Type == tokens.Illegal {
			// Illegal chars hard stop lexing.
			errs = append(errs, errors.New("illegal character: "+tok.Literal))
			break
		}

		if tok.Type == tokens.EOF {
			break
		}

		toks = append(toks, tok)
	}

	return LexResult{
		Tokens: toks,
		Errors: errs,
	}
}

// nextToken gets the next token from the source.
func (l *Lexer) nextToken() (tokens.Token, error) {
	// Skip whitespace.
	err := l.skipWhitespace()
	if err != nil {
		// Skipping whitespace can fail if we encounter an unterminated block comment.
		// In that case, we return EOF to stop lexing.
		return tokens.Token{Type: tokens.EOF}, nil
	}

	ch := l.peek()

	// End of file.
	if ch == 0 {
		return tokens.Token{Type: tokens.EOF}, nil
	}

	startLine := l.line
	startCol := l.col

	switch ch {
	case '"':
		// String literal.

		decoded, err := l.readString()
		if err != nil {
			return tokens.Token{}, err
		}

		return tokens.Token{Type: tokens.String, Literal: string(decoded), Bytes: decoded, Line: startLine, Col: startCol}, nil
	default:
		// Default scan for identifiers and keywords.
		if isIdentStart(ch) {
			start := l.pos
			for isIdentPart(l.peek()) {
				l.advance()
			}

			ident := string(l.src[start:l.pos])

			// Idents can turn into keywords, so check if this ident is a keyword.
			if kwType := tokens.LookupKeyword(ident); kwType != tokens.Ident {
				return tokens.Token{Type: kwType, Literal: ident, Line: startLine, Col: startCol}, nil
			}

			// Its a real identifier.
			return tokens.Token{Type: tokens.Ident, Literal: ident, Line: startLine, Col: startCol}, nil
		}

		// If we get here, its an illegal character.
		l.advance()
		return tokens.Token{Type: tokens.Illegal, Literal: string(ch), Line: startLine, Col: startCol}, nil
	}
}

// readString reads a string literal from the input, starting after the opening quote.
func (l *Lexer) readString() ([]byte, error) {
	l.advance() // skip opening quote
	out := make([]byte, 32)
	for {
		ch := l.peek()
		if ch == 0 {
			return nil, errors.New("unterminated string literal")
		}

		if ch == '\n' || ch == '\r' {
			return nil, errors.New("newline not allowed inside string literal")
		}

		if ch == '"' {
			l.advance() // skip closing quote
			return out, nil
		}
		l.advance()
	}
}

// skipWhitespace advances the position until it encounters a non-whitespace character.
func (l *Lexer) skipWhitespace() error {
	for {
		ch := l.peek()
		if ch == 0 {
			return nil
		}

		// Skip comment lines.
		if ch == '/' && l.peekAhead(1) == '/' {
			// Skip until end of line.
			for {
				ch = l.peek()
				if ch == '\n' || ch == 0 {
					l.advance()
					break
				}
				l.advance()
			}
			// Reset to top of loop to check for more whitespace/comments after this line comment.
			continue
		}

		// Skip block comments.
		if ch == '/' && l.peekAhead(1) == '*' {
			l.advance() // skip '/'
			l.advance() // skip '*'

			// Skip until '*/'
			for {
				ch = l.peek()
				if ch == 0 {
					return errors.New("unterminated block comment")
				}

				if ch == '*' && l.peekAhead(1) == '/' {
					l.advance() // skip '*'
					l.advance() // skip '/'
					break
				}

				l.advance()
			}
			// Reset to top of loop to check for more whitespace/comments after this block comment.
			continue
		}

		// Skip whitespace characters.
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}

	return nil
}

// isIdentStart checks if the given character can start an identifier (letter or underscore).
func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isIdentPart checks if the given character can be part of an identifier (letter, digit, or underscore).
func isIdentPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// peek returns the next character without advancing the position.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}

	return l.src[l.pos]
}

// peekAhead returns the character at the given offset from the current position without advancing.
func (l *Lexer) peekAhead(cnt int) rune {
	if l.pos+cnt >= len(l.src) {
		return 0
	}
	return l.src[l.pos+cnt]
}

// advance advances the read position by 1 rune.
func (l *Lexer) advance() {
	if l.pos < len(l.src) {
		// Update line and column tracking.
		if l.src[l.pos] == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}

		l.pos++
	}
}
