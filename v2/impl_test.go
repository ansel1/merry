package merry

import (
	"errors"
	"fmt"
	"github.com/ansel1/merry"
	"github.com/stretchr/testify/assert"
	"testing"
)

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
	e1 := New("blue")
	c1 := New("fm")
	e2 := merry.Prepend(e1, "color")
	e3 := e2.WithCause(c1)

	assert.Equal(t, e2, merry.unwrap(e3))
	assert.Equal(t, e1, merry.unwrap(e2))
}

func TestErrImpl_Is(t *testing.T) {
	e1 := merry.New("blue")
	c1 := merry.New("fm")
	e2 := merry.Prepend(e1, "color").WithCause(c1)
	e3 := merry.New("red")

	assert.True(t, merry.is(e2, e2))
	assert.True(t, merry.is(e2, e1))
	assert.True(t, merry.is(e2, c1))
	assert.False(t, merry.is(e2, e3))

	// test that it works with non-merry errors in the chain
	var e4 error = &UnwrapperError{e2}
	var e5 error = merry.Prepend(e4, "asdf")

	assert.True(t, merry.is(e5, e2))
	assert.True(t, merry.is(e4, e2))
	assert.True(t, merry.is(e5, c1))
	assert.True(t, merry.is(e5, e1))
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestErrImpl_As(t *testing.T) {
	e1 := merry.New("blue error")

	var rerr *redError
	assert.False(t, merry.as(e1, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	e2 := merry.Wrap(&rr)

	assert.True(t, merry.as(e2, &rerr))
	assert.Equal(t, &rr, rerr)

	// test that it works with non-merry errors in the chain
	w := &UnwrapperError{err: e2}
	e3 := merry.Prepend(w, "asdf")

	rerr = nil

	assert.True(t, merry.as(e3, &rerr))
	assert.Equal(t, &rr, rerr)

	rerr = nil

	assert.True(t, merry.as(w, &rerr))
	assert.Equal(t, &rr, rerr)
}

func TestErrImpl_Error(t *testing.T) {
	err := errors.New("red")

	assert.Equal(t, "red", err.Error())

	err = merry.Prepend(err, "blue")

	assert.Equal(t, "blue: red", err.Error())
}

func TestErrImpl_Format(t *testing.T) {
	e := merry.New("Hi")
	assert.Equal(t, fmt.Sprintf("%v", e), e.Error())
	assert.Equal(t, fmt.Sprintf("%s", e), e.Error())
	assert.Equal(t, fmt.Sprintf("%q", e), fmt.Sprintf("%q", e.Error()))

	// %v and %s also print the cause, if there is one
	e = merry.WithCause(e, merry.New("Bye"))
	assert.Equal(t, fmt.Sprintf("%v", e), e.Error()+": Bye")
	assert.Equal(t, fmt.Sprintf("%s", e), e.Error()+": Bye")

	// %+v should return full details, including properties registered with RegisterXXX() functions
	// and the stack.
	e = merry.WithUserMessage(e, "blue")
	e = merry.Wrap(e, SetUserMessage("blue"))
	assert.Equal(t, fmt.Sprintf("%+v", e), merry.Details(e))
}

func BenchmarkIs(b *testing.B) {
	root := merry.New("root")
	err := root
	for i := 0; i < 1000; i++ {
		err = merry.New("wrapper").WithCause(err)
		for j := 0; j < 10; j++ {
			err = merry.Prepend(err, "wrapped")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assert.True(b, merry.Is(err, root))
	}
}
