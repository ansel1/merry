package merry

import "fmt"

// Wrapper knows how to wrap errors with context information.
type Wrapper interface {
	// Wrap returns a new error, wrapping the argument, and typically adding some context information.
	// skipCallers is how many callers to skip when capturing a stack to skip to the caller of the merry
	// API surface.  It's intended to make it possible to write wrappers which capture stacktraces.  e.g.
	//
	//     func CaptureStack() Wrapper {
	//         return WrapperFunc(func(err error, skipCallers int) error {
	//             s := make([]uintptr, 50)
	//             // Callers
	//             l := runtime.Callers(2+skipCallers, s[:])
	//             return SetStack(s[:l]).Wrap(err, skipCallers + 1)
	//         })
	//    }
	Wrap(err error, skipCallers int) error
}

// WrapperFunc implements Wrapper.
type WrapperFunc func(error, int) error

// Wrap implements the Wrapper interface.
func (w WrapperFunc) Wrap(err error, callerDepth int) error {
	return w(err, callerDepth)
}

// SetValue associates a key/value pair with an error.
func SetValue(key, value interface{}) Wrapper {
	return WrapperFunc(func(err error, _ int) error {
		return Set(err, key, value)
	})
}

// SetUserMessage associates an end-user message with an error.
func SetUserMessage(msg string) Wrapper {
	return SetValue(errKeyUserMessage, msg)
}

// SetUserMessagef associates a formatted end-user message with an error.
func SetUserMessagef(format string, args ...interface{}) Wrapper {
	return SetValue(errKeyUserMessage, fmt.Sprintf(format, args...))
}

// SetHTTPCode associates an HTTP status code with an error.
func SetHTTPCode(statusCode int) Wrapper {
	return SetValue(errKeyHTTPCode, statusCode)
}

// SetStack associates a stack of caller frames with an error.  Generally, this package
// will automatically capture and associate a stack with errors which are created or
// wrapped by this package.  But this allows the caller to associate an externally
// generated stack.
func SetStack(stack []uintptr) Wrapper {
	return SetValue(errKeyStack, stack)
}

// SetFormattedStack associates a stack of pre-formatted strings describing frames of a
// stacktrace.  Generally, a formatted stack is generated from the raw []uintptr stack
// associated with the error, but a pre-formatted stack can be associated with the error
// instead, and takes precedence over the raw stack.  This is useful if pre-formatted
// stack information is coming from some other source.
func SetFormattedStack(stack []string) Wrapper {
	return SetValue(errKeyFormattedStack, stack)
}

// NoCaptureStack will suppress capturing a stack, even if StackCaptureEnabled() == true.
func NoCaptureStack() Wrapper {
	return SetValue(errKeyStack, nil)
}

// ForceCaptureStack will force a stack capture, even if StackCaptureEnabled() == false,
// or if the a stack is already attached to the error (the new stack will override the earlier
// stack).
func ForceCaptureStack() Wrapper {
	return WrapperFunc(func(err error, callerDepth int) error {
		return captureStack(err, callerDepth+1, true)
	})
}

// CaptureStack will override an earlier stack with a stack captured from the current
// call site.  If StackCaptureEnabled() == false, this is a no-op.
func CaptureStack() Wrapper {
	return WrapperFunc(func(err error, callerDepth int) error {
		return captureStack(err, callerDepth+1, StackCaptureEnabled())
	})
}

// Set wraps an error with a key/value pair.  This is the simplest form of associating
// a value with an error.  It does not capture a stacktrace, invoke hooks, or do any
// other processing.  It is mainly intended as a primitive for writing Wrapper implementations.
//
// if err is nil, returns nil.
func Set(err error, key, value interface{}) error {
	if err == nil {
		return nil
	}
	return &errImpl{
		err:   err,
		key:   key,
		value: value,
	}
}
