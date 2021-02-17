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

type merryErr struct {
	err        error
	key, value interface{}
}

// Format implements fmt.Formatter
func (e *merryErr) Format(s fmt.State, verb rune) {
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
func (e *merryErr) Error() string {
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
func (e *merryErr) Cause() error {
	return Cause(e)
}
