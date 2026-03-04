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
	Int    TokenType = "INT"
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

	PlusPlus       TokenType = "++"
	MinusMinus     TokenType = "--"
	PlusAssign     TokenType = "+="
	MinusAssign    TokenType = "-="
	AsteriskAssign TokenType = "*="
	SlashAssign    TokenType = "/="

	// Comparisons and logical.
	Bang   TokenType = "!"
	Tilde  TokenType = "~"
	Amp    TokenType = "&"
	Pipe   TokenType = "|"
	Caret  TokenType = "^"
	Shl    TokenType = "<<"
	Shr    TokenType = ">>"
	AndAnd TokenType = "&&"
	OrOr   TokenType = "||"
	EqEq   TokenType = "=="
	NotEq  TokenType = "!="
	LT     TokenType = "<"
	GT     TokenType = ">"
	LE     TokenType = "<="
	GE     TokenType = ">="

	LBrace    TokenType = "{"
	RBrace    TokenType = "}"
	LBracket  TokenType = "["
	RBracket  TokenType = "]"
	LParen    TokenType = "("
	RParen    TokenType = ")"
	Comma     TokenType = ","
	Dot       TokenType = "."
	Assign    TokenType = "="
	Semicolon TokenType = ";"
	Plus      TokenType = "+"
	Minus     TokenType = "-"
	Asterisk  TokenType = "*"
	Slash     TokenType = "/"
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
	}

	return Ident
}
