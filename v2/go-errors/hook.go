// Package go_errors provides a merry hook to integrate go-error stacktraces with merry.
// The hook will detect errors created with github.com/go-errors/errors, and translate
// it's stack into a merry stack.
package go_errors

import (
	"github.com/ansel1/merry/v2"
	"github.com/ansel1/merry/v2/internal"
)

func Install() {
	merry.AddHooks(IntegrateStacks())
}

type callerser interface{
	Callers() []uintptr
}

func IntegrateStacks() merry.Wrapper {
	return merry.WrapperFunc(func(err error, depth int) error {
		if err == nil || merry.HasStack(err) {
			return err
		}

		var c callerser

		if internal.As(err, &c) {
			if stack := c.Callers(); len(stack) > 0 {
				return merry.WithStack(stack).Wrap(err, depth)
			}
		}

		return err
	})
}