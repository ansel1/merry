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
	assert.Equal(t, fmt.Sprintf("%v", e), e.Error()+": Bye")
	assert.Equal(t, fmt.Sprintf("%s", e), e.Error()+": Bye")

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

func TestErrImpl_Cause(t *testing.T) {
	// err with no cause returns nil
	base := &errImpl{err: errors.New("boom"), key: "color", value: "red"}
	assert.Nil(t, base.Cause())

	// err with cause will return cause
	withCause := &errImpl{err: errors.New("bang"), key: errKeyCause, value: base}
	assert.EqualError(t, withCause.Cause(), "boom")

	// cause will search whole chain of errImpls
	impl2 := &errImpl{err: withCause, key: errKeyUserMessage, value:"yikes"}
	assert.EqualError(t, impl2.Cause(), "boom")

	// cause will fallback on the package Cause() function if it encounters
	// an error of a different type in the chain
	err := &UnwrapperError{err: impl2}
	impl3 := &errImpl{err: err, key: "size", value: "big"}
	assert.EqualError(t, impl3.Cause(), "boom")
}

func TestErrImpl_Value(t *testing.T) {
	// returns value if key matches
	base := &errImpl{err: errors.New("boom"), key: "color", value: "red"}
	assert.Nil(t, base.Value("size"))
	assert.Equal(t, "red", base.Value("color"))

	// will search whole chain of errImpls
	impl2 := &errImpl{err: base, key: errKeyUserMessage, value:"yikes"}
	assert.Nil(t, impl2.Value("size"))
	assert.Equal(t, "red", impl2.Value("color"))
	assert.Equal(t, "yikes", impl2.Value(errKeyUserMessage))

	// cause will fallback on the package Cause() function if it encounters
	// an error of a different type in the chain
	err := &UnwrapperError{err: impl2}
	impl3 := &errImpl{err: err, key: "size", value: "big"}
	assert.Equal(t, "big", impl3.Value("size"))
	assert.Equal(t, "red", impl3.Value("color"))
	assert.Equal(t, "yikes", impl3.Value(errKeyUserMessage))
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
	e1 := &errImpl{err: errors.New("blue"), key:"color", value:"red"}
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




