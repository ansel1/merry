package merry

import (
	"fmt"
	"io"
)

type errKey int

const (
	errKeyNone errKey = iota
	errKeyStack
	errKeyMessage
	errKeyHTTPCode
	errKeyUserMessage
	errKeyCause
	errKeyFormattedStack
)

func (e errKey) String() string {
	switch e {
	case errKeyNone:
		return "none"
	case errKeyStack:
		return "stack"
	case errKeyMessage:
		return "message"
	case errKeyHTTPCode:
		return "http status code"
	case errKeyUserMessage:
		return "user message"
	case errKeyCause:
		return "cause"
	case errKeyFormattedStack:
		return "formatted stack"
	}
	return ""
}

type errImpl struct {
	err        error
	key, value interface{}
}

// Format implements fmt.Formatter
func (e *errImpl) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, Details(e))
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// Error implements golang's error interface
// returns the message value if set, otherwise
// delegates to inner error
func (e *errImpl) Error() string {
	if verbose {
		return Details(e)
	}

	m := Message(e)
	if m == "" {
		m = UserMessage(e)
	}
	// add cause
	if c := Cause(e); c != nil {
		if ce := c.Error(); ce != "" {
			m += ": " + ce
		}
	}

	return m
}

// Cause returns the cause of the receiver, or nil if there is
// no cause, or the receiver is nil
func (e *errImpl) Cause() error {
	return Cause(e)
}

// Value returns the value associated with the specified key.  It will search
// recursively through all wrapped errors.
func (e *errImpl) Value(key interface{}) interface{} {
	v, ok, err := e.iterativeValueSearch(key)
	if ok {
		return v
	}

	return Value(err, key)
}

func (e *errImpl) iterativeValueSearch(key interface{}) (interface{}, bool, error) {
	// optimization: search using iteration first, until we get to a error
	// which isn't our internal type.  It's much faster than recursion.
	for {
		if key == e.key {
			return e.value, true, e
		}

		if n, ok := e.err.(*errImpl); ok {
			e = n
		} else {
			break
		}
	}

	// search failed.  return the most deeply wrapped error, so it can be unwrapped and searched recursively
	return nil, false, e.err
}

// Unwrap returns the next wrapped error.
func (e *errImpl) Unwrap() error {
	return e.err
}

// Is implements the new go errors.Is function.  It checks the main
// // chain of wrapped errors first, then checks the cause.
func (e *errImpl) Is(err error) bool {
	if is(e.err, err) {
		return true
	}
	if e.key == errKeyCause {
		if c, ok := e.value.(error); ok {
			return is(c, err)
		}
	}
	return false
}

// As implements the new go errors.As function.  It checks the main
// chain of wrapped errors first, then checks the cause.
func (e *errImpl) As(target interface{}) bool {
	if as(e.err, target) {
		return true
	}
	if e.key == errKeyCause {
		if c, ok := e.value.(error); ok {
			return as(c, target)
		}
	}
	return false
}
