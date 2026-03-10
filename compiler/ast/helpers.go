package ast

import (
	"github.com/MBlore/AuAu/token"
)

func TokenTypeToString(t token.TokenType) string {
	switch t {
	case token.Add:
		return "+"
	case token.Mul:
		return "*"
	case token.Sub:
		return "-"
	case token.Div:
		return "/"
	case token.EqEq:
		return "=="
	case token.NotEq:
		return "!="
	case token.Lt:
		return "<"
	case token.LtEq:
		return "<="
	case token.Gt:
		return ">"
	case token.GtEq:
		return ">="
	case token.AndAnd:
		return "&&"
	case token.OrOr:
		return "||"
	}

	return "Unknown"
}

func TypeKindToString(t TypeKind) string {
	switch t {
	case TypeInt64:
		return "int64"
	case TypeVoid:
		return "void"
	case TypeNull:
		return "null"
	case TypeInt:
		return "int"
	case TypeInt32:
		return "int32"
	case TypeInt16:
		return "int16"
	case TypeInt8:
		return "int8"
	case TypeUInt64:
		return "uint64"
	case TypeUInt32:
		return "uint32"
	case TypeUInt16:
		return "uint16"
	case TypeUInt8:
		return "uint8"
	case TypeBool:
		return "bool"
	case TypeByte:
		return "byte"
	case TypeRune:
		return "rune"
	case TypeString:
		return "string"
	}
	return "unknown"
}

func TypeToString(t *TypeRef) string {
	if t == nil {
		return "nil"
	}

	return TypeKindToString(t.Kind)
}
