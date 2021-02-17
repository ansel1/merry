package merry

import "fmt"

// Error extends the standard golang `error` interface with functions
// for attachment additional data to the error
type Error interface {
	error
	Appendf(format string, args ...interface{}) Error
	Append(msg string) Error
	Prepend(msg string) Error
	Prependf(format string, args ...interface{}) Error
	WithMessage(msg string) Error
	WithMessagef(format string, args ...interface{}) Error
	WithUserMessage(msg string) Error
	WithUserMessagef(format string, args ...interface{}) Error
	WithValue(key, value interface{}) Error
	Here() Error
	WithStackSkipping(skip int) Error
	WithHTTPCode(code int) Error
	WithCause(err error) Error
	Cause() error
	fmt.Formatter
}

// make sure merryErr implements Error
var _ Error = (*merryErr)(nil)

// return a new error with additional context
func (e *merryErr) WithValue(key, value interface{}) Error {
	if e == nil {
		return nil
	}
	return &merryErr{
		err:   e,
		key:   key,
		value: value,
	}
}

// Shorthand for capturing a new stack trace
func (e *merryErr) Here() Error {
	return HereSkipping(e, 1)
}

// return a new error with a new stack capture
func (e *merryErr) WithStackSkipping(skip int) Error {
	return HereSkipping(e, skip+1)
}

// return a new error with an http status code attached
func (e *merryErr) WithHTTPCode(code int) Error {
	if e == nil {
		return nil
	}
	return e.WithValue(errKeyHTTPCode, code)
}

// return a new error with a new message
func (e *merryErr) WithMessage(msg string) Error {
	if e == nil {
		return nil
	}
	return e.WithValue(errKeyMessage, msg)
}

// return a new error with a new formatted message
func (e *merryErr) WithMessagef(format string, a ...interface{}) Error {
	if e == nil {
		return nil
	}
	return e.WithMessage(fmt.Sprintf(format, a...))
}

// Add a message which is suitable for end users to see
func (e *merryErr) WithUserMessage(msg string) Error {
	if e == nil {
		return nil
	}
	return e.WithValue(errKeyUserMessage, msg)
}

// Add a message which is suitable for end users to see
func (e *merryErr) WithUserMessagef(format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return e.WithUserMessage(fmt.Sprintf(format, args...))
}

// Append a message after the current error message, in the format "original: new"
func (e *merryErr) Append(msg string) Error {
	if e == nil {
		return nil
	}
	return e.WithMessagef("%s: %s", Message(e), msg)
}

// Append a message after the current error message, in the format "original: new"
func (e *merryErr) Appendf(format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return e.Append(fmt.Sprintf(format, args...))
}

// Prepend a message before the current error message, in the format "new: original"
func (e *merryErr) Prepend(msg string) Error {
	if e == nil {
		return nil
	}
	return e.WithMessagef("%s: %s", msg, Message(e))
}

// Prepend a message before the current error message, in the format "new: original"
func (e *merryErr) Prependf(format string, args ...interface{}) Error {
	if e == nil {
		return nil
	}
	return e.Prepend(fmt.Sprintf(format, args...))
}

// WithCause returns an error based on the receiver, with the cause
// set to the argument.
func (e *merryErr) WithCause(err error) Error {
	if e == nil || err == nil {
		return e
	}
	return e.WithValue(errKeyCause, err)
}
