package driver

import (
	"fmt"
	"os"

	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/parser"
)

const (
	ansiReset     = "\x1b[0m"
	ansiLightBlue = "\x1b[94m"
	ansiGreen     = "\x1b[32m"
	ansiRed       = "\x1b[31m"
)

// colorize wraps text with one ANSI color code and a trailing reset code.
func colorize(text, ansiColor string) string {
	return ansiColor + text + ansiReset
}

func Run(args []string) {
	fmt.Println("========================================================================")
	fmt.Println(colorize(" AuAu Compiler v0.1.0", ansiLightBlue))
	fmt.Println(" ...because even high witches need a compiler.")
	fmt.Println("========================================================================")

	// Only support "build" for now.
	if len(args) < 3 {
		fmt.Println("Usage: auau build <filename>")
		return
	}

	command := args[1]
	if command != "build" {
		fmt.Printf("Unknown command: %s\n", command)
		return
	}

	filename := args[2]

	// File must exist.
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("File not found: %s\n", filename)
		return
	}

	fmt.Printf("Compiling %s...\n", filename)

	// Phase one: Lex.
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return
	}

	// Step one: Lex the source code into tokens.
	lexer := lexer.NewLexer(string(source))
	lexResult := lexer.Lex()

	// Report lexing errors.
	if len(lexResult.Errors) > 0 {
		for _, err := range lexResult.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}

		return
	}

	// Step two: Parse the tokens into an AST.
	parser := parser.NewParser(filename, lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) > 0 {
		for _, err := range pr.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}

		return
	}

	fmt.Println(colorize("Build successful.", ansiGreen))
}
