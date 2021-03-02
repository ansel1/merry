package merry

import (
	"errors"
	"fmt"
	"github.com/ansel1/merry/v2/internal"
	"runtime"
)

// New creates a new error, with a stack attached.  The equivalent of golang's errors.New()
func New(msg string, wrappers ...Wrapper) error {
	return WrapSkipping(errors.New(msg), 1, wrappers...)
}

// Errorf creates a new error with a formatted message and a stack.  The equivalent of golang's fmt.Errorf().
// args may contain either arguments to format, or Wrapper options, which will be applied to the error.
func Errorf(format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return WrapSkipping(fmt.Errorf(format, fmtArgs...), 1, wrappers...)
}

// Sentinel creates an error without running hooks or capturing a stack.  It is intended
// to create sentinel errors, which will be wrapped with a stack later from where the
// error is returned.  At that time, a stack will be captured and hooks will be run.
//
//     var ErrNotFound = merry.Sentinel("not found", merry.WithHTTPCode(404))
//
//     func FindUser(name string) (*User, error) {
//       // some db code which fails to find a user
//       return nil, merry.Wrap(ErrNotFound)
//     }
//
//     func main() {
//       _, err := FindUser("bob")
//       fmt.Println(errors.Is(err, ErrNotFound) // "true"
//       fmt.Println(merry.Details(err))         // stacktrace will start at the return statement
//                                               // in FindUser()
//     }
func Sentinel(msg string, wrappers ...Wrapper) error {
	return apply(errors.New(msg), 1, false, false, wrappers...)
}

// Sentinelf is like Sentinel, but takes a formatted message.  args can be a mix of
// format arguments and Wrappers.
func Sentinelf(format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return apply(fmt.Errorf(format, fmtArgs...), 1, false, false, wrappers...)
}

func splitWrappers(args []interface{}) ([]interface{}, []Wrapper) {
	var wrappers []Wrapper

	// pull out the args which are wrappers
	n := 0
	for _, arg := range args {
		if w, ok := arg.(Wrapper); ok {
			wrappers = append(wrappers, w)
		} else {
			args[n] = arg
			n++
		}
	}
	args = args[:n]

	return args, wrappers
}

// Wrap adds context to errors by applying Wrappers.  See WithXXX() functions for Wrappers supplied
// by this package.
//
// If StackCaptureEnabled is true, a stack starting at the caller will be automatically captured
// and attached to the error.  This behavior can be overridden with wrappers which either capture
// their own stacks, or suppress auto capture.
//
// If err is nil, returns nil.
func Wrap(err error, wrappers ...Wrapper) error {
	return WrapSkipping(err, 1, wrappers...)
}

// WrapSkipping is like Wrap, but the captured stacks will start `skip` frames
// further up the call stack.  If skip is 0, it behaves the same as Wrap.
func WrapSkipping(err error, skip int, wrappers ...Wrapper) error {
	return apply(err, skip+1, true, true, wrappers...)
}

// apply wraps an error with wrappers, and optionally applies hooks and ensures
// the error has a stack.  This is a low-level API intended for granular control
// over how the error is processed.
//
// todo: consider making public
func apply(err error, skip int, applyHooks, autocapture bool, wrappers ...Wrapper) error {
	if err == nil {
		return nil
	}

	if applyHooks {
		for _, h := range hooks {
			err = h.Wrap(err, skip+1)
		}
	}

	for _, w := range wrappers {
		err = w.Wrap(err, skip+1)
	}

	if autocapture {
		err = captureStack(err, skip+1, false)
	}

	return err
}

// Prepend is a convenience function for the PrependMessage wrapper.  It eases migration
// from merry v1.  It accepts a varargs of additional Wrappers.
func Prepend(err error, msg string, wrappers ...Wrapper) error {
	return WrapSkipping(err, 1, append(wrappers, PrependMessage(msg))...)
}

// Prependf is a convenience function for the PrependMessagef wrapper.  It eases migration
// from merry v1.  The args can be format arguments mixed with Wrappers.
func Prependf(err error, format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return WrapSkipping(err, 1, append(wrappers, PrependMessagef(format, fmtArgs...))...)
}

// Append is a convenience function for the AppendMessage wrapper.  It eases migration
// from merry v1.  It accepts a varargs of additional Wrappers.
func Append(err error, msg string, wrappers ...Wrapper) error {
	return WrapSkipping(err, 1, append(wrappers, AppendMessage(msg))...)
}

// Appendf is a convenience function for the AppendMessagef wrapper.  It eases migration
// from merry v1.  The args can be format arguments mixed with Wrappers.
func Appendf(err error, format string, args ...interface{}) error {
	fmtArgs, wrappers := splitWrappers(args)

	return WrapSkipping(err, 1, append(wrappers, AppendMessagef(format, fmtArgs...))...)
}

// Value returns the value for key, or nil if not set.
// If e is nil, returns nil.
func Value(err error, key interface{}) interface{} {
	for err != nil {
		if impl, ok := err.(*errImpl); ok {
			if impl.key == key {
				return impl.value
			}
			err = impl.err
		} else {
			err = internal.Unwrap(err)
		}
	}

	return nil
}

// Values returns a map of all values attached to the error
// If a key has been attached multiple times, the map will
// contain the last value mapped
// If e is nil, returns nil.
func Values(err error) map[interface{}]interface{} {
	var values map[interface{}]interface{}

	for err != nil {
		if e, ok := err.(*errImpl); ok {
			if _, ok := values[e.key]; !ok {
				if values == nil {
					values = map[interface{}]interface{}{}
				}
				values[e.key] = e.value
			}
		}
		err = internal.Unwrap(err)
	}

	return values
}

// Stack returns the stack attached to an error, or nil if one is not attached
// If e is nil, returns nil.
func Stack(err error) []uintptr {
	stack, _ := Value(err, errKeyStack).([]uintptr)
	return stack
}

// HTTPCode converts an error to an http status code.  All errors
// map to 500, unless the error has an http code attached.
// If e is nil, returns 200.
func HTTPCode(err error) int {
	if err == nil {
		return 200
	}

	code, _ := Value(err, errKeyHTTPCode).(int)
	if code == 0 {
		return 500
	}

	return code
}

// UserMessage returns the end-user safe message.  Returns empty if not set.
// If e is nil, returns "".
func UserMessage(err error) string {
	msg, _ := Value(err, errKeyUserMessage).(string)
	return msg
}

// Cause returns the cause of the argument.  If e is nil, or has no cause,
// nil is returned.
func Cause(err error) error {
	var causer *errWithCause
	if internal.As(err, &causer) {
		return causer.cause
	}
	return nil
}

// captureStack: return an error with a stack attached.  Stack will skip
// specified frames.  skip = 0 will start at caller.
// If the err already has a stack, to auto-stack-capture is disabled globally,
// this is a no-op.  Use force to override and force a stack capture
// in all cases.
func captureStack(err error, skip int, force bool) error {
	if err == nil {
		return nil
	}
	if !force && (!captureStacks || HasStack(err)) {
		return err
	}

	s := make([]uintptr, MaxStackDepth())
	length := runtime.Callers(2+skip, s[:])
	return Set(err, errKeyStack, s[:length])
}

// HasStack returns true if a stack is already attached to the err.
// If err == nil, returns false.
//
// If a stack capture was suppressed with NoCaptureStack(), this will
// still return true, indicating that stack capture processing has already
// occurred on this error.
func HasStack(err error) bool {
	for err != nil {
		if e, ok := err.(*errImpl); ok {
			if e.key == errKeyStack || e.key == errKeyFormattedStack {
				return true
			}
			err = e.err
		} else {
			err = internal.Unwrap(err)
		}
	}
	return false
}
