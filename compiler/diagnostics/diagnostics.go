package diagnostics

import (
	"fmt"

	"github.com/MBlore/AuAu/ast"
)

var sourceFile string

func SetSourceFile(file string) {
	sourceFile = file
}

func WrapError(nm ast.NodeMeta, err error) error {
	return fmt.Errorf("%s:%d:%d: %s", sourceFile, nm.Line, nm.Col, err.Error())
}

func WrapErrorAt(line, col int, err error) error {
	return fmt.Errorf("%s:%d:%d: %s", sourceFile, line, col, err.Error())
}
