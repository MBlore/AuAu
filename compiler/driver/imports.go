package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MBlore/AuAu/ast"
	"github.com/MBlore/AuAu/diagnostics"
	"github.com/MBlore/AuAu/lexer"
	"github.com/MBlore/AuAu/parser"
)

var builtPackages map[string]*ast.File

// resolveImports walks the imports tree for the given AST, building and merging imported packages as it goes.
// The passed AST is modified in place to include all the merged imports.
func resolveImports(f *ast.File) error {

	builtPackages = make(map[string]*ast.File)

	// We'll take note of the module folders as we go so we can load and compile then after.
	// First thing we should make sure of is that there are no circular imports.
	imported := make(map[string]*ast.File)
	imported[f.ModulePath] = f

	err := resolveModuleImports(f, imported)
	if err != nil {
		return err
	}

	// Merge all built packages into the main AST.
	for _, b := range builtPackages {
		f.Merge(b)
	}

	return nil
}

func resolveModuleImports(f *ast.File, imported map[string]*ast.File) error {
	// Copy the imported map so we can modify it without affecting the caller.
	importedCopy := make(map[string]*ast.File)
	for k, v := range imported {
		importedCopy[k] = v
	}

	for _, imp := range f.Imports {
		importPath := filepath.Join(f.ModulePath, imp.PackageName)
		if _, err := os.Stat(importPath); os.IsNotExist(err) {
			return fmt.Errorf("imported module not found: %s", imp.PackageName)
		}

		// If this new import has already been imported, then we have a circular import.
		if _, isImported := importedCopy[importPath]; isImported {
			return fmt.Errorf("circular import detected: %s", imp.PackageName)
		}

		// build the package.
		importedAst := buildPackage(importPath)
		if importedAst == nil {
			return fmt.Errorf("failed to build imported module: %s", imp.PackageName)
		}

		// Record the imported module.
		importedCopy[importPath] = importedAst

		// Now this module has imports, so we walk those.
		if err := resolveModuleImports(importedAst, importedCopy); err != nil {
			return err
		}
	}

	return nil
}

// buildPackage will collect all source files at the specified path and do a folder build,
// returning the merged AST for all source files.
// It caches to ensure we dont build the same package multiple times.
func buildPackage(path string) *ast.File {
	if ast, exists := builtPackages[path]; exists {
		return ast
	}

	var sourceFiles []string

	// Collect source files in the target folder.
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".au") {
			sourceFiles = append(sourceFiles, filepath.Join(path, entry.Name()))
		}
	}

	// Parse each source file and merge their ASTs.
	var mergedAst *ast.File
	for _, sourceFile := range sourceFiles {
		ast, err := parseFile(sourceFile)
		if err != nil {
			fmt.Printf("Error parsing file %s: %s\n", sourceFile, err)
			return nil
		}

		if mergedAst == nil {
			mergedAst = ast
		} else {
			mergedAst.Merge(ast)
		}
	}

	builtPackages[path] = mergedAst

	return mergedAst
}

// parseFile takes a single filename and returns the compiled AST.
func parseFile(path string) (*ast.File, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %s", err)
	}

	diagnostics.SetSourceFile(path)

	lx := lexer.NewLexer(string(source))
	lexResult := lx.Lex()

	if len(lexResult.Errors) > 0 {
		for _, err := range lexResult.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}

		return nil, fmt.Errorf("file failed to parse")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %s", err)
	}

	filename := filepath.Base(absPath)

	parser := parser.NewParser(filename, lexResult.Tokens)
	pr := parser.Parse()
	if len(pr.Errors) > 0 {
		for _, err := range pr.Errors {
			fmt.Println(colorize("error ", ansiRed) + err.Error())
		}
		return nil, fmt.Errorf("file failed to parse")
	}

	pr.File.ModulePath = filepath.Dir(absPath)

	return pr.File, nil
}
