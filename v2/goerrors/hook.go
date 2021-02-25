// Package goerrors provides a merry hook to integrate go-error stacktraces with merry.
// The hook will detect errors created with github.com/goerrors/errors, and translate
// it's stack into a merry stack.
package goerrors

import (
	"github.com/ansel1/merry/v2"
	"github.com/ansel1/merry/v2/internal"
)

// Install installs IntegrateStacks as a merry hook.
func Install() {
	merry.AddHooks(IntegrateStacks())
}

type callerser interface {
	Callers() []uintptr
}

// IntegrateStacks searches the error chain for errors implementing
// callerser and returning a non-empty stack.  It attaches the stack
// to the error using merry.
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
