package merry

import (
	"fmt"
	"github.com/ansel1/merry/v2/internal"
	"io"
	"strings"
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
	errKeyForceCapture
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
	case errKeyForceCapture:
		return "force stack capture"
	default:
		return ""
	}
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
		io.WriteString(s, msgWithCauses(e))
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

func msgWithCauses(err error) string {
	messages := make([]string, 0, 5)

	for err != nil {
		if ce := err.Error(); ce != "" {
			messages = append(messages, ce)
		}
		err = Cause(err)
	}

	return strings.Join(messages, ": ")
}

// Error implements golang's error interface
// returns the message value if set, otherwise
// delegates to inner error
func (e *errImpl) Error() string {
	if e.key == errKeyMessage {
		if s, ok := e.value.(string); ok {
			return s
		}
	}
	return e.err.Error()
}

// String implements fmt.Stringer
func (e *errImpl) String() string {
	return e.Error()
}

// Unwrap returns the next wrapped error.
func (e *errImpl) Unwrap() error {
	return e.err
}

// Is implements the new go errors.Is function.  Returns
// true if is(cause, target)
func (e *errImpl) Is(target error) bool {
	if e.key == errKeyCause {
		if c, ok := e.value.(error); ok {
			return internal.Is(c, target)
		}
	}
	return false
}

// As implements the new go errors.As function.  Returns
// true if as(cause, target)
func (e *errImpl) As(target interface{}) bool {
	if e.key == errKeyCause {
		if c, ok := e.value.(error); ok {
			return internal.As(c, target)
		}
	}
	return false
}
