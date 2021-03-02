package merry

import (
	"fmt"
	"github.com/ansel1/merry/v2/internal"
	"io"
	"reflect"
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
	format(s, verb, e)
}

func format(s fmt.State, verb rune, err error) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, Details(err))
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, msgWithCauses(err))
	case 'q':
		fmt.Fprintf(s, "%q", err.Error())
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

type errWithCause struct {
	err   error
	cause error
}

func (e *errWithCause) Unwrap() error {
	var nextErr error
	if e1, ok := e.err.(*errWithCause); ok {
		nextErr = e1.err
	} else {
		nextErr = internal.Unwrap(e.err)
	}
	if nextErr == nil {
		return e.cause
	}
	return &errWithCause{err: nextErr, cause: e.cause}
}

func (e *errWithCause) String() string {
	return e.Error()
}

func (e *errWithCause) Error() string {
	return e.err.Error()
}

func (e *errWithCause) Format(f fmt.State, verb rune) {
	format(f, verb, e)
}

func (e *errWithCause) Is(target error) bool {
	// This does most of what errors.Is() does, by delegating
	// to the nested error.  But it does not use Unwrap to recurse
	// any further.  This just compares target with next error in the stack.
	isComparable := reflect.TypeOf(target).Comparable()
	if isComparable && e.err == target {
		return true
	}

	// if the next error is another errWithCause, don't bother calling its
	// Is() implementation.  Let errors.Is() fallback to Unwrap(), which
	// will skip through to the next nested error.
	if _, ok := e.err.(*errWithCause); ok {
		return false
	}
	if x, ok := e.err.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}
	return false
}

func (e *errWithCause) As(target interface{}) bool {
	// This does most of what errors.As() does, by delegating
	// to the nested error.  But it does use Unwrap to recurse
	// any further. This just compares target with next error in the stack.
	val := reflect.ValueOf(target)
	typ := val.Type()
	targetType := typ.Elem()
	if reflect.TypeOf(e.err).AssignableTo(targetType) {
		val.Elem().Set(reflect.ValueOf(e.err))
		return true
	}

	// if the next error is another errWithCause, don't bother calling its
	// As() implementation.  Let errors.As() fallback to Unwrap(), which
	// will skip through to the next nested error.
	if _, ok := e.err.(*errWithCause); ok {
		return false
	}
	if x, ok := e.err.(interface{ As(interface{}) bool }); ok && x.As(target) {
		return true
	}
	return false
}
