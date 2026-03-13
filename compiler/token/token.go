package token

// TokenType identifies the lexical class for one token.
type TokenType string

const (
	Illegal TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"
	Ident   TokenType = "IDENT"
	Package TokenType = "PACKAGE"
	Import  TokenType = "IMPORT"

	// Literals
	Number TokenType = "NUMBER"
	String TokenType = "STRING"
	Float  TokenType = "FLOAT"
	Rune   TokenType = "RUNE"

	// Keywords and literals.
	Void      TokenType = "VOID"
	IntKw     TokenType = "INT_KW"
	Int8Kw    TokenType = "INT8_KW"
	Int16Kw   TokenType = "INT16_KW"
	Int32Kw   TokenType = "INT32_KW"
	Int64Kw   TokenType = "INT64_KW"
	UIntKw    TokenType = "UINT_KW"
	UInt8Kw   TokenType = "UINT8_KW"
	UInt16Kw  TokenType = "UINT16_KW"
	UInt32Kw  TokenType = "UINT32_KW"
	UInt64Kw  TokenType = "UINT64_KW"
	ByteKw    TokenType = "BYTE_KW"
	RuneKw    TokenType = "RUNE_KW"
	StringKw  TokenType = "STRING_KW"
	BoolKw    TokenType = "BOOL_KW"
	Float32Kw TokenType = "FLOAT32_KW"
	Float64Kw TokenType = "FLOAT64_KW"
	FloatKw   TokenType = "FLOAT_KW"

	True     TokenType = "TRUE"
	False    TokenType = "FALSE"
	Null     TokenType = "NULL"
	Print    TokenType = "PRINT"
	If       TokenType = "IF"
	Else     TokenType = "ELSE"
	While    TokenType = "WHILE"
	For      TokenType = "FOR"
	Break    TokenType = "BREAK"
	Continue TokenType = "CONTINUE"
	Return   TokenType = "RETURN"
	Extern   TokenType = "EXTERN"
	Struct   TokenType = "STRUCT"

	// Compound assignment and increment/decrement.
	AddAdd    TokenType = "++"
	SubSub    TokenType = "--"
	AddAssign TokenType = "+="
	SubAssign TokenType = "-="
	MulAssign TokenType = "*="
	DivAssign TokenType = "/="

	// Bitwise.
	Bang  TokenType = "!"
	Tilde TokenType = "~"
	Amp   TokenType = "&"
	Pipe  TokenType = "|"
	Caret TokenType = "^"
	Shl   TokenType = "<<"
	Shr   TokenType = ">>"

	// Logical.
	AndAnd TokenType = "&&"
	OrOr   TokenType = "||"

	// Equality.
	EqEq  TokenType = "=="
	NotEq TokenType = "!="

	// Relational.
	Lt   TokenType = "<"
	Gt   TokenType = ">"
	LtEq TokenType = "<="
	GtEq TokenType = ">="

	LBrace    TokenType = "{"
	RBrace    TokenType = "}"
	LBracket  TokenType = "["
	RBracket  TokenType = "]"
	LParen    TokenType = "("
	RParen    TokenType = ")"
	Comma     TokenType = ","
	Dot       TokenType = "."
	Equals    TokenType = "="
	Semicolon TokenType = ";"
	Add       TokenType = "+"
	Sub       TokenType = "-"
	Mul       TokenType = "*"
	Div       TokenType = "/"
	Percent   TokenType = "%"
)

// Token represents one parsed token from the source code.
type Token struct {
	Type TokenType
	// The literal value of the token, as it appears in the source code.
	Literal string
	// Strings and runes are stored as byte slices to preserve escape sequences.
	Bytes []byte

	// The line and column where the token was found.
	Line int
	Col  int
}

// LookupKeyword checks if the given identifier is a keyword and returns the appropriate token type.
func LookupKeyword(ident string) TokenType {
	switch ident {
	case "package":
		return Package
	case "void":
		return Void
	case "int":
		return IntKw
	case "return":
		return Return
	case "if":
		return If
	case "else":
		return Else
	case "while":
		return While
	case "for":
		return For
	case "break":
		return Break
	case "continue":
		return Continue
	case "extern":
		return Extern
	case "struct":
		return Struct
	case "import":
		return Import
	case "uint8":
		return UInt8Kw
	case "uint16":
		return UInt16Kw
	case "uint32":
		return UInt32Kw
	case "uint64":
		return UInt64Kw
	case "uint":
		return UIntKw
	case "int8":
		return Int8Kw
	case "int16":
		return Int16Kw
	case "int32":
		return Int32Kw
	case "int64":
		return Int64Kw
	case "byte":
		return ByteKw
	case "rune":
		return RuneKw
	case "string":
		return StringKw
	case "bool":
		return BoolKw
	case "float32":
		return Float32Kw
	case "float64":
		return Float64Kw
	case "float":
		return FloatKw
	case "true":
		return True
	case "false":
		return False
	case "null":
		return Null
	}

	return Ident
}
