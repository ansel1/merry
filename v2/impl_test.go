package merry

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrWithValue_Format(t *testing.T) {
	// %v and %s print the same as err.Error() if there is no cause
	err := &errWithValue{err: errors.New("Hi")}
	assert.IsType(t, &errWithValue{}, err)
	assert.Equal(t, fmt.Sprintf("%v", err), err.Error())
	assert.Equal(t, fmt.Sprintf("%s", err), err.Error())

	// %q returns err.Error() as a golang literal
	assert.Equal(t, fmt.Sprintf("%q", err), fmt.Sprintf("%q", err.Error()))

	// if there is a cause in the chain, include it.
	err = Wrap(err, WithCause(New("Bye")), WithValue("color", "red")).(*errWithValue)
	assert.Equal(t, fmt.Sprintf("%v", err), "Hi: Bye")
	assert.Equal(t, fmt.Sprintf("%s", err), "Hi: Bye")

	// %+v should return full details, including properties registered with RegisterXXX() functions
	// and the stack.
	err = Wrap(err, WithUserMessage("blue")).(*errWithValue)
	assert.Equal(t, fmt.Sprintf("%+v", err), Details(err))
}

func TestErrWithCause_Format(t *testing.T) {
	// %v and %s also print the cause, if there is one
	err := &errWithCause{err: New("Hi"), cause: New("Bye")}
	// make sure we have an errWithCause here, to ensure that it also implements
	// fmt.Formatter
	assert.IsType(t, &errWithCause{}, err)
	assert.Equal(t, fmt.Sprintf("%v", err), "Hi: Bye")
	assert.Equal(t, fmt.Sprintf("%s", err), "Hi: Bye")

	// %q returns err.Error() as a golang literal
	assert.Equal(t, fmt.Sprintf("%q", err), fmt.Sprintf("%q", err.Error()))

	// %+v should return full details, including properties registered with RegisterXXX() functions
	// and the stack.
	assert.Equal(t, fmt.Sprintf("%+v", err), Details(err))
}

func TestErrWithValue_Error(t *testing.T) {
	err := &errWithValue{err: errors.New("red")}

	assert.Equal(t, "red", err.Error())

	err = Wrap(err, WithMessage("blue")).(*errWithValue)

	assert.Equal(t, "blue", err.Error())
}

func TestErrWithCause_Error(t *testing.T) {
	err := &errWithCause{err: errors.New("blue"), cause: errors.New("red")}
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

func TestErrWithValue_Unwrap(t *testing.T) {
	e1 := &errWithValue{err: errors.New("blue"), key: "color", value: "red"}
	assert.EqualError(t, e1.Unwrap(), "blue")
}

func TestErrWithCause_Unwrap(t *testing.T) {
	topErr := Apply(errors.New("blue"), WithMessage("green"))
	err := &errWithCause{err: topErr, cause: errors.New("red")}

	// unwrapping the layers should return green, then blue, then the cause (red).
	// first unwrap should return blue
	var layers []string
	var unwrapped error = err
	for unwrapped != nil {
		layers = append(layers, unwrapped.Error())
		unwrapped = errors.Unwrap(unwrapped)
	}
	// first error in chain should be the errWithCause wrapping
	// the green error.  Unwrapping should produce an errWithCause
	// wrapping the blue error,  Unwrapping again should produce the cause.
	assert.Equal(t, []string{"green", "blue", "red"}, layers)

	// if another cause is attached, it should override the older cause.  The older
	// cause should no longer be returned by Unwrap.
	unwrapped = &errWithCause{err: err, cause: errors.New("yellow")}
	layers = nil
	for unwrapped != nil {
		layers = append(layers, unwrapped.Error())
		unwrapped = errors.Unwrap(unwrapped)
	}
	assert.Equal(t, []string{"green", "blue", "yellow"}, layers)
}

func TestErrWithValue_String(t *testing.T) {
	err := New("blue")
	assert.Equal(t, "blue", err.(*errWithValue).String())
	assert.Equal(t, "red", Wrap(err, WithMessage("red")).(*errWithValue).String())
}

func TestErrWithCause_String(t *testing.T) {
	assert.Equal(t, "blue", (&errWithCause{err: errors.New("blue")}).String())
}

func TestIs(t *testing.T) {
	// an error is all the errors it wraps
	e1 := New("blue")
	e2 := Wrap(e1, WithHTTPCode(5))
	assert.True(t, errors.Is(e2, e1))
	assert.False(t, errors.Is(e1, e2))

	// is works through other unwrapper implementations
	e3 := &UnwrapperError{err: e2}
	e4 := Wrap(e3, WithUserMessage("hi"))
	assert.True(t, errors.Is(e4, e3))
	assert.True(t, errors.Is(e4, e2))
	assert.True(t, errors.Is(e4, e1))

	// an error is also any of the causes
	rootCause := errors.New("ioerror")
	rootCause1 := Wrap(rootCause)
	outererr := New("failed", WithCause(rootCause1))
	outererr1 := Wrap(outererr, WithUserMessage("sorry!"))

	assert.True(t, errors.Is(outererr1, outererr))
	assert.True(t, errors.Is(outererr1, rootCause1))
	assert.True(t, errors.Is(outererr1, rootCause))

	// but only the latest cause
	newCause := errors.New("new cause")
	outererr1 = Wrap(outererr1, WithCause(newCause))
	assert.ErrorIs(t, outererr1, newCause)
	assert.NotErrorIs(t, outererr1, rootCause)
	assert.NotErrorIs(t, outererr1, rootCause1)
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestAs(t *testing.T) {
	err := New("blue error")

	// as will find matching errors in the chain
	var rerr *redError
	assert.False(t, errors.As(err, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	err = Wrap(&rr)

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	rerr = nil

	// test that it works with non-merry errors in the chain
	err = &UnwrapperError{err: err}
	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	err = Wrap(err, PrependMessage("asdf"))

	rerr = nil

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	// will search causes as well
	err = New("boom", WithCause(err))

	rerr = nil

	assert.True(t, errors.As(err, &rerr))
	assert.Equal(t, &rr, rerr)

	// but only the latest cause
	err = Wrap(err, WithCause(errors.New("new cause")))
	assert.False(t, errors.As(err, &rerr))
}
