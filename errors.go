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
	return WrapSkipping(errors.New(msg), 1, SetUserMessage(msg))
}

// UserErrorf is like UserError, but uses fmt.Sprintf()
func UserErrorf(format string, args ...interface{}) Error {
	msg := fmt.Sprintf(format, args...)
	return WrapSkipping(errors.New(msg), 1, SetUserMessage(msg))
}

// Wrap turns the argument into a merry.Error.  If the argument already is a
// merry.Error, this is a no-op.
// If e == nil, return nil
func Wrap(err error, wrappers ...Wrapper) Error {
	return WrapSkipping(err, 1, wrappers...)
}

// WrapSkipping turns the error arg into a merry.Error if the arg is not
// already a merry.Error.
// If e is nil, return nil.
// If a merry.Error is created by this call, the stack captured will skip
// `skip` frames (0 is the call site of `WrapSkipping()`)
func WrapSkipping(err error, skip int, wrappers ...Wrapper) Error {
	for _, w := range wrappers {
		err = w.Wrap(err, skip+1)
	}
	return captureStack(err, skip+1, false)
}

// WithValue adds a context an error.  If the key was already set on e,
// the new value will take precedence.
// If e is nil, returns nil.
func WithValue(err error, key, value interface{}) Error {
	return WrapSkipping(err, 1, SetValue(key, value))
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
func Values(e error) map[interface{}]interface{} {
	if e == nil {
		return nil
	}
	var values map[interface{}]interface{}
	for {
		w, ok := e.(*errImpl)
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
func Here(err error) Error {
	return captureStack(err, 1, StackCaptureEnabled())
}

// HereSkipping returns an error with a new stacktrace, at the call site
// of HereSkipping() - skip frames.
func HereSkipping(err error, skip int) Error {
	return captureStack(err, skip+1, StackCaptureEnabled())
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
	return WrapSkipping(e, 1, SetHTTPCode(code))
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

// WithCause returns an error based on the first argument, with the cause
// set to the second argument.  If e is nil, returns nil.
func WithCause(err error, cause error) Error {
	return WrapSkipping(err, 1, SetCause(cause))
}

// WithMessage returns an error with a new message.
// The resulting error's Error() method will return
// the new message.
// If e is nil, returns nil.
func WithMessage(err error, msg string) Error {
	return WrapSkipping(err, 1, SetMessage(msg))
}

// WithMessagef is the same as WithMessage(), using fmt.Sprintf().
func WithMessagef(err error, format string, args ...interface{}) Error {
	return WrapSkipping(err, 1, SetMessagef(format, args...))
}

// WithUserMessage adds a message which is suitable for end users to see.
// If e is nil, returns nil.
func WithUserMessage(err error, msg string) Error {
	return WrapSkipping(err, 1, SetUserMessage(msg))
}

// WithUserMessagef is the same as WithMessage(), using fmt.Sprintf()
func WithUserMessagef(err error, format string, args ...interface{}) Error {
	return WrapSkipping(err, 1, SetUserMessagef(format, args...))
}

// Append a message after the current error message, in the format "original: new".
// If e == nil, return nil.
func Append(err error, msg string) Error {
	return WrapSkipping(err, 1, AppendMessage(msg))
}

// Appendf is the same as Append, but uses fmt.Sprintf().
func Appendf(err error, format string, args ...interface{}) Error {
	return WrapSkipping(err, 1, AppendMessagef(format, args...))
}

// Prepend a message before the current error message, in the format "new: original".
// If e == nil, return nil.
func Prepend(err error, msg string) Error {
	return WrapSkipping(err, 1, PrependMessage(msg))
}

// Prependf is the same as Prepend, but uses fmt.Sprintf()
func Prependf(err error, format string, args ...interface{}) Error {
	return WrapSkipping(err, 1, PrependMessagef(format, args...))
}

// Is is equivalent to errors.Is, but tests against multiple targets.
//
// merry.Is(err1, err2, err3) == errors.Is(err1, err2) || errors.Is(err1, err3)
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
		w, ok := e.(*errImpl)
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
		if merr, ok := err.(*errImpl); ok {
			return merr
		}
		// wrap just to return the correct type.  We need to return a Error
		// to accommodate the chainable API
		return &errImpl{
			err: err,
		}
	}

	s := make([]uintptr, MaxStackDepth)
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
