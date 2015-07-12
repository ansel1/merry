package richerrors

// The richerrors package augments standard golang errors with stacktraces
// and http status codes.
// It also providers a way to override an error's message, while preserving
// the ability to compare error values equality, the common go idiom
// for changing for specific types of errors.  So you can define exported error values
// in your package, like:
//
//     var InvalidInputs = errors.New("Bad inputs")
//
// ...return copies of that error from your function, with more information in the message,
// and a stacktrace captured at the callsite of Extend:
//
//     return richerrors.Extend(InvalidInputs).WithMessagef("Bad inputs: %v", inputs)
//
// ...and callers can still compare the returned error to the exported error value:
//
//     if richerrors.Is(err, InvalidInputs) {
//
import (
	"errors"
	"fmt"
	goerr "github.com/go-errors/errors"
	"runtime"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

type (
	richError struct {
		err      error
		stack    []uintptr
		message  string
		httpCode int
	}

	// Something which has a call stack
	Stacker interface {
		Stack() []uintptr
	}

	// Something which has an HTTP status code
	HTTPCoder interface {
		HTTPCode() int
	}

	// Something which wraps an error
	Wrapper interface {
		// Underlying returns the underlying error, or nil
		// if there is none.
		Underlying() error
	}

	// An error with a call stack, an HTTP status code, an underlying
	// error, and methods to override the underlying error's message
	RichError interface {
		error
		Stacker
		HTTPCoder
		Wrapper
		WithHTTPCode(code int) RichError
		WithMessage(msg string) RichError
		WithMessagef(format string, a ...interface{}) RichError
	}
)

// Create a new RichError.  The equivalent of golang's errors.New()
func New(msg string) RichError {
	return Wrap(errors.New(msg), 1)
}

// Create a new RichError with a formatted message.  The equivalent of golang's fmt.Errorf()
func Errorf(format string, a ...interface{}) RichError {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Turn any error into a RichError.
// If e is already a RichError, this is a no-op.
// This should be used when handling an error from another
// library.  You want to capture a stacktrace if one was not
// already captured.  But if e already had a stacktrace
// attached, you want to preserve it, so it is as close
// to the site of the original error creation as possible.
// if skip is 0, and Wrap does capture a new stacktrace, the
// stacktrace will be captured from the callsite of Wrap().
// if skip is > 0, the stacktrace callsite will go up the call
// stack, just like runtime.Caller()
func Wrap(e error, skip int) RichError {
	if re, ok := e.(RichError); ok {
		return re
	}
	return &richError{
		err:   e,
		stack: captureStack(skip + 1),
	}
}

// Create a new error with extends another error.
// A new stack is captured at the call site of Extend()
// `Is(Extend(e), e)` will be true
func Extend(e error) RichError {
	return &richError{
		err:   e,
		stack: captureStack(1),
	}
}

// Check whether e is equal or extends the original.
func Is(e error, original error) bool {
	for {
		if e == original {
			return true
		}
		w, ok := e.(Wrapper)
		if !ok {
			return false
		}
		e = w.Underlying()
	}
}

// Convert an error to an http status code.  All errors
// map to 500, unless the error implements HTTPCoder.
func HTTPCode(e error) int {
	if he, ok := e.(HTTPCoder); ok {
		return he.HTTPCode()
	}
	return 500
}

// Return the innermost underlying error.
// Only useful in advanced cases, like if you need to
// cast the underlying error to some type to get
// additional information from it.
func Unwrap(e error) error {
	for {
		if w, ok := e.(Wrapper); ok {
			e = w.Underlying()
		} else {
			break
		}
	}
	return e
}

// Returns "unknown" if e has no stacktrace
func Location(e error) (file string, line int) {
	if e, ok := e.(Stacker); ok {
		s := e.Stack()
		if len(s) > 0 {
			sf := goerr.NewStackFrame(s[0])
			return sf.File, sf.LineNumber
		}
	}
	return "unknown", 0
}

func captureStack(skip int) []uintptr {
	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return stack[:length]
}

// implements golang's error interface
// returns Message field if set, otherwise
// delegates to inner error
func (e *richError) Error() string {
	if e.message != "" {
		return e.message
	}
	return e.err.Error()
}

// implement Wrapper interface
func (e *richError) Underlying() error {
	return e.err
}

// implement HTTPCoder interface
func (e *richError) HTTPCode() int {
	if e.httpCode > 0 {
		return e.httpCode
	}
	return HTTPCode(e.Underlying())
}

func (e *richError) Stack() []uintptr {
	if e.stack == nil {
		if e, ok := e.err.(Stacker); ok {
			return e.Stack()
		}
	}
	return e.stack
}

func (e *richError) WithHTTPCode(code int) RichError {
	return &richError{
		err:      e,
		httpCode: code,
	}
}

func (e *richError) WithMessage(msg string) RichError {
	return &richError{
		err:     e,
		message: msg,
	}
}

func (e *richError) WithMessagef(format string, a ...interface{}) RichError {
	return &richError{
		err:     e,
		message: fmt.Sprintf(format, a...),
	}
}
