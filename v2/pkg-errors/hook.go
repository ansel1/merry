// Package go_errors provides a merry hook to integrate pkg/errors stacktraces with merry.
// The hook will detect errors created with github.com/pkg/errors, and translate
// it's stack into a merry stack.
package pkg_errors

import (
	"github.com/ansel1/merry/v2"
	"github.com/ansel1/merry/v2/internal"
	"github.com/pkg/errors"
)

func Install() {
	merry.AddHooks(IntegrateStacks())
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func IntegrateStacks() merry.Wrapper {
	return merry.WrapperFunc(func(err error, depth int) error {
		var s stackTracer

		if err != nil && !merry.HasStack(err) && internal.As(err, &s) {
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
