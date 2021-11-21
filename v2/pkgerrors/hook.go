// Package pkgerrors provides a merry hook to integrate pkg/errors stacktraces with merry.
// The hook will detect errors created with github.com/pkg/errors, and translate
// it's stack into a merry stack.
package pkgerrors

import (
	errors2 "errors"
	"github.com/ansel1/merry/v2"
	"github.com/pkg/errors"
)

// Install installs IntegrateStacks() as a merry hook.
func Install() {
	merry.AddHooks(IntegrateStacks())
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// IntegrateStacks searches the error chain for errors created by
// github.com/pkg/errors, which have a stack attached.  The stack
// is attached to the merry error.
func IntegrateStacks() merry.Wrapper {
	return merry.WrapperFunc(func(err error, depth int) error {
		var s stackTracer

		if err != nil && !merry.HasStack(err) && errors2.As(err, &s) {
			if frames := s.StackTrace(); len(frames) > 0 {
				stack := make([]uintptr, len(frames))
				for i := range frames {
					stack[i] = uintptr(frames[i])
				}
				return merry.WithStack(stack).Wrap(err, depth)
			}
		}

		return err
	})
}
