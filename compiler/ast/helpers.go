package ast

import "github.com/MBlore/AuAu/token"

func TokenTypeToString(t token.TokenType) string {
	switch t {
	case token.Plus:
		return "+"
	case token.Asterisk:
		return "*"
	case token.Minus:
		return "-"
	case token.Slash:
		return "/"
	}

	return "Unknown"
}
