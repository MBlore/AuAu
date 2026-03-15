package parser

import (
	"strings"
	"testing"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/parser/validate"
	"github.com/MBlore/AuAu/token"
)

func TestSuccessParse(t *testing.T) {
	code := `package main`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if pr.File == nil {
		t.Errorf("Expected a SourceFile, got nil")
	}

	if pr.File.PackageName != "main" {
		t.Errorf("Expected package name 'main', got '%s'", pr.File.PackageName)
	}
}

func TestPackageFirstTokenError(t *testing.T) {
	code := "test"
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 1 || !strings.Contains(pr.Errors[0].Error(), "package declaration must be the first statement") {
		t.Errorf("Expected 1 error about package declaration, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestPackageFormat(t *testing.T) {
	code := "package \"main\""

	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()

	parser := NewParser("test.auau", lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 1 || !strings.Contains(pr.Errors[0].Error(), "expected package name") {
		t.Errorf("Expected 1 error about package declaration, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestPeakAheadReturnsEOF(t *testing.T) {
	code := `package main`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)

	if parser.peekAhead(100).Type != token.EOF {
		t.Errorf("Expected peekAhead(100) to return EOF, got %v", parser.peekAhead(100))
	}
}

func TestPeakReturnsEOF(t *testing.T) {
	code := `package main`
	lexer := lexer.NewLexer(code)
	lexResult := lexer.Lex()
	parser := NewParser("test.auau", lexResult.Tokens)

	// Advance to the end of the tokens.
	for i := 0; i < len(lexResult.Tokens); i++ {
		parser.advance()
	}

	if parser.peek().Type != token.EOF {
		t.Errorf("Expected peek() at end of tokens to return EOF, got %v", parser.peek())
	}
}

func TestReadOneFunc(t *testing.T) {
	input := `package main

	void main() {
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if pr.File == nil {
		t.Errorf("Expected a File, got nil")
	}

	if len(pr.File.Functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(pr.File.Functions))
	}

	mainFunc := pr.File.Functions[0]
	if mainFunc.Name != "main" {
		t.Errorf("Expected function name 'main', got '%s'", mainFunc.Name)
	}
}

func TestReadTwoFuncs(t *testing.T) {
	input := `package main

	void main() {
	}

	int Foo() {
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.File.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(pr.File.Functions))
	}

	mainFunc := pr.File.Functions[0]
	if mainFunc.Name != "main" {
		t.Errorf("Expected first function name 'main', got '%s'", mainFunc.Name)
	}
	if mainFunc.IsPublic != false {
		t.Errorf("Expected 'main' function to be private, got IsPublic=%v", mainFunc.IsPublic)
	}
	if mainFunc.ReturnType.Kind != ast.TypeVoid {
		t.Errorf("Expected 'main' function return type to be 'void', got %v", mainFunc.ReturnType.Kind)
	}

	fooFunc := pr.File.Functions[1]
	if fooFunc.Name != "Foo" {
		t.Errorf("Expected second function name 'Foo', got '%s'", fooFunc.Name)
	}
	if fooFunc.IsPublic != true {
		t.Errorf("Expected 'Foo' function to be public, got IsPublic=%v", fooFunc.IsPublic)
	}
	if fooFunc.ReturnType.Kind != ast.TypeInt {
		t.Errorf("Expected 'Foo' function return type to be 'int', got %v", fooFunc.ReturnType.Kind)
	}
}

func TestAssignmentParsing(t *testing.T) {
	input := `package main
	void main() {
		int a = 1 + 2
		int b = 1 + 2 + 3
		int c = a + b
		int d = (1) + (2)
		int e = (1 + 2) + 3
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()
	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestAssignmentParsingNested(t *testing.T) {
	input := `package main
	void main() {
		int a = 1 + (2 * -3) - 4 / 2
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()
	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestVariableTypesParsing(t *testing.T) {
	input := `package main
	void main() {
		int a = 1

		uint8 b = 255
		uint16 c = 65535
		uint32 d = 4294967295
		uint64 e = 18446744073709551615
		
		int8 f = -128
		int16 g = -32768
		int32 h = -2147483648
		int64 i = -9223372036854775808

		byte j = 255
		rune k = 1114111
		bool l = true
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	// Check parsed variable types match what we expect.
	mainFunc := pr.File.Functions[0]
	expectedTypes := []ast.TypeKind{
		ast.TypeInt, ast.TypeUInt8, ast.TypeUInt16, ast.TypeUInt32, ast.TypeUInt64,
		ast.TypeInt8, ast.TypeInt16, ast.TypeInt32, ast.TypeInt64,
		ast.TypeByte, ast.TypeRune, ast.TypeBool,
	}

	for i, stmt := range mainFunc.Body.Stmts {
		decl, ok := stmt.(*ast.VarDeclStmt)
		if !ok {
			t.Errorf("Expected VarDeclStmt, got %T", stmt)
			continue
		}
		if decl.Type.Kind != expectedTypes[i] {
			t.Errorf("Expected type %v, got %v", expectedTypes[i], decl.Type.Kind)
		}
	}
}

func TestParsesExterns(t *testing.T) {
	input := `package main

	extern int printf(string format)

	void main() { }`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()
	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if len(pr.File.Externs) != 1 {
		t.Errorf("Expected 1 extern function, got %d", len(pr.File.Externs))
	}
}

func TestParseStructDecl(t *testing.T) {
	input := `package main

	struct Point {
		x int
		y int
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()
	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	if len(pr.File.Structs) != 1 {
		t.Errorf("Expected 1 struct declaration, got %d", len(pr.File.Structs))
		return
	}

	pointStruct := pr.File.Structs[0]
	if pointStruct.Name != "Point" {
		t.Errorf("Expected struct name 'Point', got '%s'", pointStruct.Name)
	}

	if len(pointStruct.Fields) != 2 {
		t.Errorf("Expected 2 fields in struct, got %d", len(pointStruct.Fields))
		return
	}

	if pointStruct.Fields[0].Name != "x" || pointStruct.Fields[0].Type.Kind != ast.TypeInt {
		t.Errorf("Expected first field to be 'x int', got '%s %v'", pointStruct.Fields[0].Name, pointStruct.Fields[0].Type)
	}

	if pointStruct.Fields[1].Name != "y" || pointStruct.Fields[1].Type.Kind != ast.TypeInt {
		t.Errorf("Expected second field to be 'y int', got '%s %v'", pointStruct.Fields[1].Name, pointStruct.Fields[1].Type)
	}
}

func TestStructUsage(t *testing.T) {
	input := `package main
	struct Point {
		x int
		y int
	}

	void main() {
		Point p
		p.x = 10
	}

	void PrintPoint(Point pt) {
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestNestedStructs(t *testing.T) {
	input := `package main
	struct C {
		value int
	}

	struct B {
		c C
	}

	struct A {
		b B
	}

	void main() {
		A a
		a.b.c.value = 42
		print(a.b.c.value)
	}`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Fatalf("Expected 0 lexer errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Fatalf("Expected 0 parser errors, got %d: %v", len(pr.Errors), pr.Errors)
	}
}

func TestDuplicateStructNamesFailValidation(t *testing.T) {
	input := `package main
	struct Point {
		x int
	}

	struct Point {
		y int
	}
`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Fatalf("Expected 0 lexer errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Fatalf("Expected 0 parse errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	validateErrors := validate.Validate(pr.File)
	if len(validateErrors) != 1 {
		t.Fatalf("Expected 1 validation error, got %d: %v", len(validateErrors), validateErrors)
	}

	if !strings.Contains(validateErrors[0].Error(), "duplicate struct name: Point") {
		t.Fatalf("Expected duplicate struct error, got %v", validateErrors[0])
	}
}

func TestUnknownCustomTypesFailValidation(t *testing.T) {
	input := `package main
	struct Point {
		next Missing
	}

	extern void logPoint(Missing value)

	Missing makePoint(Missing input) {
		Missing local
		return local
	}
`

	lexer := lexer.NewLexer(input)
	result := lexer.Lex()

	if len(result.Errors) != 0 {
		t.Fatalf("Expected 0 lexer errors, got %d: %v", len(result.Errors), result.Errors)
	}

	parser := NewParser("test.auau", result.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) != 0 {
		t.Fatalf("Expected 0 parse errors, got %d: %v", len(pr.Errors), pr.Errors)
	}

	validateErrors := validate.Validate(pr.File)
	if len(validateErrors) < 5 {
		t.Fatalf("Expected at least 5 validation errors, got %d: %v", len(validateErrors), validateErrors)
	}

	allErrors := make([]string, 0, len(validateErrors))
	for _, err := range validateErrors {
		allErrors = append(allErrors, err.Error())
	}

	joined := strings.Join(allErrors, "\n")

	for _, expected := range []string{
		"unknown type in struct field Point.next: Missing",
		"unknown parameter type for extern logPoint parameter value: Missing",
		"unknown return type for function makePoint: Missing",
		"unknown parameter type for function makePoint parameter input: Missing",
		"unknown variable type for local: Missing",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("Expected error %q in %s", expected, joined)
		}
	}
}
