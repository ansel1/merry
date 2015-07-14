package merry

// The merry package augments standard golang errors with stacktraces
// and other context information.
//
// You can add any context information to an error with `e = merry.WithValue(e, "code", 12345)`
// You can retrieve that value with `v, _ := merry.Value(e, "code").(int)`
//
// Any error augmented like this will automatically get a stacktrace attached, if it doesn't have one
// already.  If you just want to add the stacktrace, use `Wrap(e)`
//
// It also providers a way to override an error's message:
//
//     var InvalidInputs = errors.New("Bad inputs")
//
// `Here()` captures a new stacktrace, and WithMessagef() sets a new error message:
//
//     return merry.Here(InvalidInputs).WithMessagef("Bad inputs: %v", inputs)
//
// Errors are immutable.  All functions and methods which add context return new errors.
// But errors can still be compared to the originals with `Is()`
//
//     if merry.Is(err, InvalidInputs) {
//
// Functions which add context to errors have equivalent methods on *Error, to allow
// convenient chaining:
//
//     return merry.New("Invalid body").WithHTTPCode(400)

import (
	"errors"
	"fmt"
	"runtime"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

type errorProperty string

const (
	stack    errorProperty = "stack"
	message                = "message"
	httpCode               = "http status code"
)

type Error struct {
	err        error
	key, value interface{}
}

// Create a new error, with a stack attached.  The equivalent of golang's errors.New()
func New(msg string) *Error {
	return WrapSkipping(errors.New(msg), 1)
}

// Create a new error with a formatted message and a stack.  The equivalent of golang's fmt.Errorf()
func Errorf(format string, a ...interface{}) *Error {
	return WrapSkipping(fmt.Errorf(format, a...), 1)
}

// Cast e to *Error, or wrap e in a new *Error with stack
func Wrap(e error) *Error {
	return WrapSkipping(e, 1)
}

// Cast e to *Error, or wrap e in a new *Error with stack
// Skip `skip` frames (0 is the call site of `WrapSkipping()`)
func WrapSkipping(e error, skip int) *Error {
	if re, ok := e.(*Error); ok {
		return re
	}
	return &Error{
		err:   e,
		key:   stack,
		value: captureStack(skip + 1),
	}
}

// Add a context an error.  If the key was already set on e,
// the new value will take precedence.
func WithValue(e error, key, value interface{}) *Error {
	return Wrap(e).WithValue(key, value)
}

// Return the value for key, or nil if not set
func Value(e error, key interface{}) interface{} {
	for {
		m, ok := e.(*Error)
		if !ok {
			return nil
		}
		if m.key == key {
			return m.value
		}
		e = m.err
	}
}

// Attach a new stack to the error, at the call site of Here().
// Useful when returning copies of exported package errors
func Here(e error) *Error {
	// optimization: only capture the stack once, since its expensive
	if re, ok := e.(*Error); ok {
		return re.WithStackSkipping(1)
	}
	return WrapSkipping(e, 1)
}

// Return the stack attached to an error, or nil if one is not attached
func Stack(e error) []uintptr {
	stack, _ := Value(e, stack).([]uintptr)
	return stack
}

// Return an error with an http code attached.
func WithHTTPCode(e error, code int) *Error {
	return Wrap(e).WithHTTPCode(code)
}

// Convert an error to an http status code.  All errors
// map to 500, unless the error has an http code attached.
func HTTPCode(e error) int {
	code, _ := Value(e, httpCode).(int)
	if code == 0 {
		return 500
	}
	return code
}

// Override the message of error.
// The resulting error's Error() method will return
// the new message
func WithMessage(e error, msg string) *Error {
	return Wrap(e).WithValue(message, msg)
}

// Same as WithMessage(), using fmt.Sprint()
func WithMessagef(e error, format string, a ...interface{}) *Error {
	return Wrap(e).WithMessagef(format, a...)
}

// Check whether e is equal to or wraps the original, at any depth
func Is(e error, original error) bool {
	for {
		if e == original {
			return true
		}
		w, ok := e.(*Error)
		if !ok {
			return false
		}
		e = w.err
	}
}

// Return the innermost underlying error.
// Only useful in advanced cases, like if you need to
// cast the underlying error to some type to get
// additional information from it.
func Unwrap(e error) error {
	for {
		w, ok := e.(*Error)
		if !ok {
			return e
		}
		e = w.err
	}
	return e
}

func captureStack(skip int) []uintptr {
	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return stack[:length]
}

// implements golang's error interface
// returns the message value if set, otherwise
// delegates to inner error
func (e *Error) Error() string {
	m, _ := Value(e, message).(string)
	if m == "" {
		return Unwrap(e).Error()
	}
	return m
}

// return a new error with additional context
func (e *Error) WithValue(key, value interface{}) *Error {
	return &Error{
		err:   e,
		key:   key,
		value: value,
	}
}

// Shorthand for capturing a new stack trace
func (e *Error) Here() *Error {
	return e.WithStackSkipping(1)
}

// return a new error with a new stack capture
func (e *Error) WithStackSkipping(skip int) *Error {
	return &Error{
		err:   e,
		key:   stack,
		value: captureStack(skip + 1),
	}
}

// return a new error with an http status code attached
func (e *Error) WithHTTPCode(code int) *Error {
	return e.WithValue(httpCode, code)
}

// return a new error with a new message
func (e *Error) WithMessage(msg string) *Error {
	return e.WithValue(message, msg)
}

// return a new error with a new formatted message
func (e *Error) WithMessagef(format string, a ...interface{}) *Error {
	return e.WithMessage(fmt.Sprintf(format, a...))
}
