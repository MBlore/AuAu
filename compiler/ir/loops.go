package ir

import (
	"fmt"

	"github.com/MBlore/AuAu/ast"
)

type loopContext struct {
	breakBlock    *Block
	continueBlock *Block
}

func (l *LoweringContext) pushLoop(breakBlock, continueBlock *Block) {
	l.loops = append(l.loops, loopContext{
		breakBlock:    breakBlock,
		continueBlock: continueBlock,
	})
}

func (l *LoweringContext) popLoop() {
	if len(l.loops) == 0 {
		panic("popLoop called with empty loop stack")
	}

	l.loops = l.loops[:len(l.loops)-1]
}

func (l *LoweringContext) currentLoop() (loopContext, bool) {
	if len(l.loops) == 0 {
		return loopContext{}, false
	}

	return l.loops[len(l.loops)-1], true
}

func (l *LoweringContext) emitForLoop(s *ast.ForStmt) error {
	// For loops are a bit more complex since they can have an init statement, a condition, and a post statement.
	initBlock := l.builder.NewBlock("for_init")
	condBlock := l.builder.NewBlock("for_cond")
	bodyBlock := l.builder.NewBlock("for_body")
	postBlock := l.builder.NewBlock("for_post")
	endBlock := l.builder.NewBlock("for_end")

	l.pushLoop(endBlock, postBlock)
	defer l.popLoop()

	// Emit the init statement if it exists, then jump to the condition.
	if s.Init != nil {
		l.builder.Jump(initBlock)
		l.builder.SetBlock(initBlock)
		if err := l.emitStmt(s.Init); err != nil {
			return fmt.Errorf("invalid init statement in for loop: %w", err)
		}

		l.builder.Jump(condBlock)
	} else {
		l.builder.Jump(condBlock)
	}

	// Emit the condition block. If there is no condition, we will just jump to the body.
	l.builder.SetBlock(condBlock)
	if s.Cond != nil {
		condVal, err := l.emitExpr(s.Cond)
		if err != nil {
			return fmt.Errorf("invalid condition expression in for loop: %w", err)
		}

		l.builder.Branch(condVal, bodyBlock, endBlock)
	} else {
		l.builder.Jump(bodyBlock)
	}

	// Emit the body block.
	l.builder.SetBlock(bodyBlock)

	if err := l.emitBlock(s.Body); err != nil {
		return fmt.Errorf("invalid body in for loop: %w", err)
	}

	// If the body doesn't end with a return or branch, add a jump to the post block.
	if !blockTerminated(l.builder.CurrentBlock()) {
		l.builder.Jump(postBlock)
	}

	// Emit the post statement if it exists, then jump back to the condition.
	l.builder.SetBlock(postBlock)
	if s.Post != nil {
		if err := l.emitStmt(s.Post); err != nil {
			return fmt.Errorf("invalid post statement in for loop: %w", err)
		}
	}

	l.builder.Jump(condBlock)
	l.builder.SetBlock(endBlock)

	return nil
}
