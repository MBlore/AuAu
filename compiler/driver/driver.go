package driver

import (
	"fmt"
	"os"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/backend/nasm/x64"
	"github.com/MBlore/AuAu/diagnostics"
	"github.com/MBlore/AuAu/ir"
	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/parser"
	"github.com/MBlore/AuAu/parser/validate"
	"github.com/MBlore/AuAu/token"
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
	if len(args) < 2 {
		fmt.Println("Usage: auau build <path>")
		fmt.Println("\nBuild a source file or all source files in the given path.")
		fmt.Println("\nExamples:")
		fmt.Println("  auau build main.au")
		fmt.Println("  auau build ./src")
		fmt.Println("  auau build")
		return
	}

	command := args[1]
	if command != "build" {
		fmt.Printf("Unknown command: %s\n", command)
		return
	}

	if len(args) < 3 {
		// No path provided, default to current directory.
		args = append(args, ".")
	}

	if info, err := os.Stat(args[2]); err == nil && info.IsDir() {
		// Do folder build.
		buildFolder(args[2])
		fmt.Println(colorize("Build successful.", ansiGreen))
		return
	}

	buildFile(args[2])

	fmt.Println(colorize("Build successful.", ansiGreen))
}

// buildFolder compiles all .au files in the given folder.
func buildFolder(folderPath string) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		return
	}

	mergedAst := &ast.File{PackageName: "main"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if entry.Name()[len(entry.Name())-3:] == ".au" {
			// For every source file we find, we part in to AST, and merge it with a global AST,
			// that we'll use to do a single build.
			source, err := os.ReadFile(folderPath + "/" + entry.Name())
			if err != nil {
				fmt.Printf("Error reading file: %s\n", err)
				return
			}

			lx := lexer.NewLexer(string(source))
			lexResult := lx.Lex()
			if len(lexResult.Errors) > 0 {
				for _, err := range lexResult.Errors {
					fmt.Println(colorize("error ", ansiRed) + err.Error())
				}
				return
			}

			parser := parser.NewParser(entry.Name(), lexResult.Tokens)
			pr := parser.Parse()
			if len(pr.Errors) > 0 {
				for _, err := range pr.Errors {
					fmt.Println(colorize("error ", ansiRed) + err.Error())
				}
				return
			}

			// Merge the parsed AST into the global AST.
			if err := mergedAst.Merge(pr.File); err != nil {
				fmt.Println(colorize("error ", ansiRed) + err.Error())
				return
			}
		}
	}

	compileAst(mergedAst)
}

func buildFile(filename string) {
	// File must exist.
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("File not found: %s\n", filename)
		return
	}

	diagnostics.SetSourceFile(filename)

	fmt.Printf("Compiling %s...\n", filename)

	// Load source code from file.
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err)
		return
	}

	// Step 1: Lex the source code into tokens.
	lx := lexer.NewLexer(string(source))
	lexResult := lx.Lex()

	// Report lexing errors.
	if len(lexResult.Errors) > 0 {
		for _, err := range lexResult.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}

		return
	}

	// Print the tokens.
	tokStr := token.PrintTokens(lexResult.Tokens)

	err = os.WriteFile("tokens.txt", []byte(tokStr), 0644)
	if err != nil {
		fmt.Printf("Error writing tokens to file: %s\n", err)
		return
	}

	// Step 2: Parse the tokens into an AST.
	parser := parser.NewParser(filename, lexResult.Tokens)
	pr := parser.Parse()

	if len(pr.Errors) > 0 {
		for _, err := range pr.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}

		return
	}

	// Print the AST to file.
	astPrint := ast.NewAstPrinter(pr.File)
	astStr := astPrint.Print()

	err = os.WriteFile("ast.txt", []byte(astStr), 0644)
	if err != nil {
		fmt.Printf("Error writing AST to file: %s\n", err)
		return
	}

	compileAst(pr.File)
}

// compileAst takes a fully parsed AST, validates it and compiles it all the way down to assembly.
func compileAst(f *ast.File) {
	// Semantic checks.
	validationErrors := validate.Validate(f)
	if len(validationErrors) > 0 {
		for _, err := range validationErrors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}
		return
	}

	// Step 3: Lower the AST to IR.
	irProgram, err := ir.CompileFile(f)
	if err != nil {
		fmt.Printf("Error lowering to IR: %s\n", err)
		return
	}

	// TODO: Optimize the IR here.

	// Step 4: Compile the IR to assembly.
	err = x64.Compile("out.asm", irProgram)
	if err != nil {
		fmt.Printf("Error compiling to assembly: %s\n", err)
		return
	}
}
