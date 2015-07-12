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
// and a stacktrace captured at the callsite of Copy:
//
//     return richerrors.Copy(InvalidInputs).WithMessagef("Bad inputs: %v", inputs)
//
// ...and callers can still compare the returned error to the exported error value:
//
//     if richerrors.Is(err, InvalidInputs) {
//
import (
	goerr "github.com/go-errors/errors"
	"runtime"
	"errors"
	"fmt"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

type (
	richError struct {
		err error
		stack  []uintptr
		frames []goerr.StackFrame
		message string
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
		// Message returns the top level error message,
		// not including the message from the underlying
		// error.
		Message() string

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
		err: e,
		stack: captureStack(skip+1),
	}
}

// Create a RichError copy of e.
// If e not a RichError, this function will
// work just like Wrap().
// If e is a RichError, create a new RichError, copying
// the underlying error, code, and override message (if
// any) from e, but capture a new stacktrace.
func Copy(e error) RichError {
	err := Unwrap(e)
	re := &richError{
		err:err,
		stack: captureStack(1),
		httpCode: HTTPCode(e),
	}
	if w, ok := e.(Wrapper); ok {
		re.message = w.Message()
	}
	return re
}

// Create a new error with extends another error.  `Is()` will
// return true for all errors which extend the parent error.
// Currently, this is just a synonym for Copy
func Extend(e error) RichError {
	return Copy(e)
}

// Check whether e is equal to or a copy of original.
func Is(e error, original error) bool {
	return e == original || Unwrap(e) == Unwrap(original)
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
func (e *richError) Message() string {
	return e.message
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
	return 500
}

func (e *richError) Stack() []uintptr {
	return e.stack
}

func (e *richError) WithHTTPCode(code int) RichError {
	e.httpCode = code
	return e
}

func (e *richError) WithMessage(msg string) RichError {
	e.message = msg
	return e
}

func (e *richError) WithMessagef(format string, a ...interface{}) RichError {
	e.message = fmt.Sprintf(format, a...)
	return e
}