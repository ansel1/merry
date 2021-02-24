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

	return WrapSkipping(fmt.Errorf(format, args...), 1, wrappers...)
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
	if err == nil {
		return nil
	}

	for _, h := range hooks {
		err = h.Wrap(err, skip +1)
	}

	for _, w := range wrappers {
		err = w.Wrap(err, skip+1)
	}

	return captureStack(err, skip+1, false)
}

// Value returns the value for key, or nil if not set.
// If e is nil, returns nil.
func Value(err error, key interface{}) interface{} {
	var valuer interface{ Value(interface{}) interface{} }
	if internal.As(err, &valuer) {
		return valuer.Value(key)
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
	var causer interface{ Cause() error }
	if internal.As(err, &causer) {
		return causer.Cause()
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
			continue
		}
		err = internal.Unwrap(err)
	}
	return false
}