package main

import (
	"fmt"
	"os"

	"github.com/MBlore/AuAu/lexer"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: auau build <filename>")
		return
	}

	// Only support "build" for now.
	command := os.Args[1]
	if command != "build" {
		fmt.Printf("Unknown command: %s\n", command)
		return
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: auau build <filename>")
		return
	}

	filename := os.Args[2]

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

	lexer := lexer.NewLexer(string(source))
	lexResult := lexer.Lex()

	fmt.Printf("Lexed %d tokens.\n", len(lexResult.Tokens))
}
