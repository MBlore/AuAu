package lexer

import (
	"fmt"
	"strings"

	"github.com/MBlore/AuAu/token"
)

type TokenPrinter struct {
	tokens []token.Token
}

func NewTokenPrinter(tokens []token.Token) *TokenPrinter {
	return &TokenPrinter{tokens: tokens}
}

func (tp *TokenPrinter) Print() string {
	var b strings.Builder

	for _, tok := range tp.tokens {
		fmt.Fprintf(&b, "[%d:%d] %s\n", tok.Line, tok.Col, tok.Type)
	}

	return b.String()
}
