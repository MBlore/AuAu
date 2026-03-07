package token

import (
	"fmt"
	"strings"
)

func PrintTokens(tokens []Token) string {
	var b strings.Builder

	for _, tok := range tokens {
		fmt.Fprintf(&b, "[%d:%d] %s (%s)\n", tok.Line, tok.Col, tok.Type, tok.Literal)
	}

	return b.String()
}
