package merry

import (
	"errors"
	"fmt"
	"github.com/ansel1/merry/v2/internal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrImpl_Format(t *testing.T) {
	// %v and %s print the same as err.Error() if there is no cause
	e := New("Hi")
	assert.Equal(t, fmt.Sprintf("%v", e), e.Error())
	assert.Equal(t, fmt.Sprintf("%s", e), e.Error())

	// %q returns e.Error() as a golang literal
	assert.Equal(t, fmt.Sprintf("%q", e), fmt.Sprintf("%q", e.Error()))

	// %v and %s also print the cause, if there is one
	e = New("Bye", WithCause(e))
	assert.Equal(t, fmt.Sprintf("%v", e), "Bye: Hi")
	assert.Equal(t, fmt.Sprintf("%s", e), "Bye: Hi")

	// %+v should return full details, including properties registered with RegisterXXX() functions
	// and the stack.
	e = Wrap(e, WithUserMessage("blue"))
	assert.Equal(t, fmt.Sprintf("%+v", e), Details(e))
}

func TestErrImpl_Error(t *testing.T) {
	err := errors.New("red")

	assert.Equal(t, "red", err.Error())

	err = Wrap(err, WithMessage("blue"))

	assert.Equal(t, "blue", err.Error())
}

// UnwrapperError is a simple error implementation that wraps another error, and implements `Unwrap() error`.
// It is used to test when errors not created by this package are inserted in the chain of wrapped errors.
type UnwrapperError struct {
	err error
}

func (w *UnwrapperError) Error() string {
	return w.err.Error()
}

func (w *UnwrapperError) Unwrap() error {
	return w.err
}

func TestErrImpl_Unwrap(t *testing.T) {
	e1 := &errImpl{err: errors.New("blue"), key: "color", value: "red"}
	assert.EqualError(t, e1.Unwrap(), "blue")
}

func TestErrImpl_Is(t *testing.T) {
	// an error is all the errors it wraps
	e1 := New("blue")
	e2 := Wrap(e1, WithHTTPCode(5))
	assert.True(t, internal.Is(e2, e1))
	assert.False(t, internal.Is(e1, e2))

	// is works through other unwrapper implementations
	e3 := &UnwrapperError{err: e2}
	e4 := Wrap(e3, WithUserMessage("hi"))
	assert.True(t, internal.Is(e4, e3))
	assert.True(t, internal.Is(e4, e2))
	assert.True(t, internal.Is(e4, e1))

	// an error is also any of the causes
	rootCause := errors.New("ioerror")
	rootCause1 := Wrap(rootCause)
	outererr := New("failed", WithCause(rootCause1))
	outererr1 := Wrap(outererr, WithUserMessage("sorry!"))

	assert.True(t, internal.Is(outererr1, outererr))
	assert.True(t, internal.Is(outererr1, rootCause1))
	assert.True(t, internal.Is(outererr1, rootCause))
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestErrImpl_As(t *testing.T) {
	e1 := New("blue error")

	// as will find matching errors in the chain
	var rerr *redError
	assert.False(t, internal.As(e1, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	e2 := Wrap(&rr)

	assert.True(t, internal.As(e2, &rerr))
	assert.Equal(t, &rr, rerr)

	// test that it works with non-merry errors in the chain
	w := &UnwrapperError{err: e2}
	e3 := Wrap(w, Prepend("asdf"))

	rerr = nil

	assert.True(t, internal.As(e3, &rerr))
	assert.Equal(t, &rr, rerr)

	rerr = nil

	assert.True(t, internal.As(w, &rerr))
	assert.Equal(t, &rr, rerr)
}

func TestErrImpl_String(t *testing.T) {
	assert.Equal(t, "blue", New("blue").(*errImpl).String())
}
