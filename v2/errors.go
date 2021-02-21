package merry

import (
	"errors"
	"fmt"
	"runtime"
)

// New creates a new error, with a stack attached.  The equivalent of golang's errors.New()
func New(msg string) error {
	return WrapSkipping(errors.New(msg), 1)
}

// Errorf creates a new error with a formatted message and a stack.  The equivalent of golang's fmt.Errorf()
func Errorf(format string, a ...interface{}) error {
	return WrapSkipping(fmt.Errorf(format, a...), 1)
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

	for _, w := range wrappers {
		err = w.Wrap(err, skip+1)
	}
	return captureStack(err, skip+1, false)
}

// Value returns the value for key, or nil if not set.
// If e is nil, returns nil.
func Value(err error, key interface{}) interface{} {
	var valuer interface{ Value(interface{}) interface{} }
	if as(err, &valuer) {
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
		err = errors.Unwrap(err)
	}

	return values
}

// Stack returns the stack attached to an error, or nil if one is not attached
// If e is nil, returns nil.
func Stack(e error) []uintptr {
	stack, _ := Value(e, errKeyStack).([]uintptr)
	return stack
}

// HTTPCode converts an error to an http status code.  All errors
// map to 500, unless the error has an http code attached.
// If e is nil, returns 200.
func HTTPCode(e error) int {
	if e == nil {
		return 200
	}

	code, _ := Value(e, errKeyHTTPCode).(int)
	if code == 0 {
		return 500
	}

	return code
}

// UserMessage returns the end-user safe message.  Returns empty if not set.
// If e is nil, returns "".
func UserMessage(e error) string {
	msg, _ := Value(e, errKeyUserMessage).(string)
	return msg
}

// Cause returns the cause of the argument.  If e is nil, or has no cause,
// nil is returned.
func Cause(e error) error {
	var causer interface{ Cause() error }
	if as(e, &causer) {
		return causer.Cause()
	}

	return nil
}

// RootCause returns the innermost cause of the argument (i.e. the last
// error in the cause chain)
func RootCause(err error) error {
	for {
		cause := Cause(err)
		if cause == nil {
			return err
		}
		err = cause
	}
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
	if !force && (!captureStacks || hasStack(err)) {
		return err
	}

	s := make([]uintptr, MaxStackDepth())
	length := runtime.Callers(2+skip, s[:])
	return &errImpl{
		err:   err,
		key:   errKeyStack,
		value: s[:length],
	}
}

func hasStack(err error) bool {
	for err != nil {
		if e, ok := err.(*errImpl); ok {
			if e.key == errKeyStack || e.key == errKeyFormattedStack {
				return true
			}
			err = e.err
			continue
		}
		err = errors.Unwrap(err)
	}
	return false
}