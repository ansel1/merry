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
//
// merry.Errors also implement fmt.Formatter, similar to github.com/pkg/errors.
//
//     fmt.Sprintf("%+v", e) == merry.Details(e)
//
// pkg/errors Cause() interface is not implemented (yet).
import (
	"errors"
	"fmt"
	"runtime"
)

// New creates a new error, with a stack attached.  The equivalent of golang's errors.New()
func New(msg string) Error {
	return WrapSkipping(errors.New(msg), 1)
}

// Errorf creates a new error with a formatted message and a stack.  The equivalent of golang's fmt.Errorf()
func Errorf(format string, a ...interface{}) Error {
	return WrapSkipping(fmt.Errorf(format, a...), 1)
}

// UserError creates a new error with a message intended for display to an
// end user.
func UserError(msg string) Error {
	return WrapSkipping(errors.New(""), 1).WithUserMessage(msg)
}

// UserErrorf is like UserError, but uses fmt.Sprintf()
func UserErrorf(format string, a ...interface{}) Error {
	return WrapSkipping(errors.New(""), 1).WithUserMessagef(format, a...)
}

// Wrap turns the argument into a merry.Error.  If the argument already is a
// merry.Error, this is a no-op.
// If e == nil, return nil
func Wrap(e error) Error {
	return captureStack(e, 1, false)
}

// WrapSkipping turns the error arg into a merry.Error if the arg is not
// already a merry.Error.
// If e is nil, return nil.
// If a merry.Error is created by this call, the stack captured will skip
// `skip` frames (0 is the call site of `WrapSkipping()`)
func WrapSkipping(e error, skip int) Error {
	return captureStack(e, skip+1, false)
}

// WithValue adds a context an error.  If the key was already set on e,
// the new value will take precedence.
// If e is nil, returns nil.
func WithValue(e error, key, value interface{}) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithValue(key, value)
}

// Value returns the value for key, or nil if not set.
// If e is nil, returns nil.
func Value(e error, key interface{}) interface{} {
	for {
		switch m := e.(type) {
		case nil:
			return nil
		case *merryErr:
			if m.key == key {
				return m.value
			}
			e = m.err
		default:
			return nil
		}
	}
}

// Values returns a map of all values attached to the error
// If a key has been attached multiple times, the map will
// contain the last value mapped
// If e is nil, returns nil.
func Values(e error) map[interface{}]interface{} {
	if e == nil {
		return nil
	}
	var values map[interface{}]interface{}
	for {
		w, ok := e.(*merryErr)
		if !ok {
			return values
		}
		if values == nil {
			values = make(map[interface{}]interface{}, 1)
		}
		if _, ok := values[w.key]; !ok {
			values[w.key] = w.value
		}
		e = w.err
	}
}

// Here returns an error with a new stacktrace, at the call site of Here().
// Useful when returning copies of exported package errors.
// If e is nil, returns nil.
func Here(e error) Error {
	return captureStack(e, 1, true)
}

// HereSkipping returns an error with a new stacktrace, at the call site
// of HereSkipping() - skip frames.
func HereSkipping(e error, skip int) Error {
	return captureStack(e, skip+1, true)
}

// Stack returns the stack attached to an error, or nil if one is not attached
// If e is nil, returns nil.
func Stack(e error) []uintptr {
	stack, _ := Value(e, errKeyStack).([]uintptr)
	return stack
}

// WithHTTPCode returns an error with an http code attached.
// If e is nil, returns nil.
func WithHTTPCode(e error, code int) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithHTTPCode(code)
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
	if e == nil {
		return ""
	}
	msg, _ := Value(e, errKeyUserMessage).(string)
	return msg
}

// Message returns just the error message.  It is equivalent to
// Error() when Verbose is false.
// The behavior of Error() is (pseudo-code):
//
//     if verbose
//       Details(e)
//     else
//       Message(e) || UserMessage(e)
//
// If e is nil, returns "".
func Message(e error) string {
	if e == nil {
		return ""
	}
	m, _ := Value(e, errKeyMessage).(string)
	if m == "" {
		m = Unwrap(e).Error()
	}
	return m
}

// Cause returns the cause of the argument.  If e is nil, or has no cause,
// nil is returned.
func Cause(e error) error {
	if e == nil {
		return nil
	}
	c, _ := Value(e, errKeyCause).(error)
	return c
}

// RootCause returns the innermost cause of the argument (i.e. the last
// error in the cause chain)
func RootCause(e error) error {
	if e == nil {
		return e
	}
	for {
		c := Cause(e)
		if c == nil {
			break
		} else {
			e = c
		}
	}
	return e
}

// WithCause returns an error based on the first argument, with the cause
// set to the second argument.  If e is nil, returns nil.
func WithCause(e error, cause error) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithCause(cause)
}

// WithMessage returns an error with a new message.
// The resulting error's Error() method will return
// the new message.
// If e is nil, returns nil.
func WithMessage(e error, msg string) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithValue(errKeyMessage, msg)
}

// WithMessagef is the same as WithMessage(), using fmt.Sprintf().
func WithMessagef(e error, format string, a ...interface{}) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithMessagef(format, a...)
}

// WithUserMessage adds a message which is suitable for end users to see.
// If e is nil, returns nil.
func WithUserMessage(e error, msg string) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithUserMessage(msg)
}

// WithUserMessagef is the same as WithMessage(), using fmt.Sprintf()
func WithUserMessagef(e error, format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).WithUserMessagef(format, args...)
}

// Append a message after the current error message, in the format "original: new".
// If e == nil, return nil.
func Append(e error, msg string) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).Append(msg)
}

// Appendf is the same as Append, but uses fmt.Sprintf().
func Appendf(e error, format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).Appendf(format, args...)
}

// Prepend a message before the current error message, in the format "new: original".
// If e == nil, return nil.
func Prepend(e error, msg string) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).Prepend(msg)
}

// Prependf is the same as Prepend, but uses fmt.Sprintf()
func Prependf(e error, format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return WrapSkipping(e, 1).Prependf(format, args...)
}

// Is checks whether e is equal to or wraps the original, at any depth.
// If e == nil, return false.
// This is useful if your package uses the common golang pattern of
// exported error constants.  If your package exports an ErrEOF constant,
// which is initialized like this:
//
//     var ErrEOF = errors.New("End of file error")
//
// ...and your user wants to compare an error returned by your package
// with ErrEOF:
//
//     err := urpack.Read()
//     if err == urpack.ErrEOF {
//
// ...the comparison will fail if the error has been wrapped by merry
// at some point.  Replace the comparison with:
//
//     if merry.Is(err, urpack.ErrEOF) {
//
// Causes
//
// Is will also return true if any of the originals is in the cause chain
// of e.  For example:
//
//     e1 := merry.New("base error")
//     e2 := merry.New("library error")
//     // e2 was caused by e1
//     e3 := merry.WithCause(e1, e2)
//     merry.Is(e3, e2)  // yes it is, because e3 is based on e2
//     merry.Is(e3, e1)  // yes it is, because e1 was a cause of e3
//
func Is(e error, originals ...error) bool {
	for _, o := range originals {
		if is(e, o) {
			return true
		}
	}
	return false
}

// Unwrap returns the innermost underlying error.
// Only useful in advanced cases, like if you need to
// cast the underlying error to some type to get
// additional information from it.
// If e == nil, return nil.
func Unwrap(e error) error {
	if e == nil {
		return nil
	}
	for {
		w, ok := e.(*merryErr)
		if !ok {
			return e
		}
		e = w.err
	}
}

// captureStack: return an error with a stack attached.  Stack will skip
// specified frames.  skip = 0 will start at caller.
// If the err already has a stack, to auto-stack-capture is disabled globally,
// this is a no-op.  Use force to override and force a stack capture
// in all cases.
func captureStack(err error, skip int, force bool) Error {
	if err == nil {
		return nil
	}
	if !force && (!captureStacks || hasStack(err)) {
		if merr, ok := err.(*merryErr); ok {
			return merr
		}
		// wrap just to return the correct type.  We need to return a *merryErr
		// to accommodate the chainable API
		return &merryErr{
			err: err,
		}
	}

	s := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, s[:])
	return &merryErr{
		err:   err,
		key:   errKeyStack,
		value: s[:length],
	}
}

func hasStack(err error) bool {
	for err != nil {
		if e, ok := err.(*merryErr); ok {
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
