package merry

import (
	"errors"
	"github.com/ansel1/merry/v2/internal"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	assert.EqualError(t, err, "bang")
	f, l := Location(err)
	assert.Contains(t, f, "errors_test.go")
	assert.Equal(t, rl+1, l)

	// New accepts wrapper options
	err = New("boom", WithUserMessage("blue"))
	assert.EqualError(t, err, "boom")
	assert.Equal(t, "blue", UserMessage(err))
}

func TestErrorf(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := Errorf("boom: %s", "uh-oh")
	assert.EqualError(t, err, "boom: uh-oh")
	f, l := Location(err)
	assert.Contains(t, f, "errors_test.go")
	assert.Equal(t, rl+1, l)

	// New accepts wrapper options
	err = Errorf("%s %s %s", "red", WithUserMessage("orange"), "blue", WithHTTPCode(5), "black")
	assert.EqualError(t, err, "red blue black")
	assert.Equal(t, "orange", UserMessage(err))
	assert.Equal(t, 5, HTTPCode(err))
}

func TestWrap(t *testing.T) {
	// capture a stack
	ogerr := errors.New("boom")
	_, _, rl, _ := runtime.Caller(0)
	err := Wrap(ogerr)
	f, l := Location(err)
	assert.Contains(t, f, "errors_test.go")
	assert.Equal(t, rl+1, l)

	// new error should wrap the old error
	assert.True(t, internal.Is(err, ogerr))

	// wrap accepts wrapper args
	err = Wrap(err, WithUserMessage("hi"), WithHTTPCode(6))
	assert.Equal(t, "hi", UserMessage(err))
	assert.Equal(t, 6, HTTPCode(err))

	// wrap should only capture the stack once, even if non-merry errors are in the chain
	// between the top and where the stack is
	err = &UnwrapperError{err}
	err = Wrap(err, WithHTTPCode(55))
	count := 0
	err1 := err
	for err1 != nil {
		if impl, ok := err.(*errImpl); ok {
			if impl.key == errKeyStack {
				count++
			}
		}
		err1 = internal.Unwrap(err1)
	}

	// wrapping nil -> nil
	assert.Nil(t, Wrap(nil))
}

func TestWrapSkipping(t *testing.T) {
	ogerr := errors.New("boom")
	var err error
	_, _, rl, _ := runtime.Caller(0)
	func() {
		err = WrapSkipping(ogerr, 1)
	}()
	f, l := Location(err)
	assert.Contains(t, f, "errors_test.go")
	// the skip arg should make the stack start at the line where the anonymous function is
	// called, rather than the line inside the function
	assert.Equal(t, rl+3, l)

	// new error should wrap the old error
	assert.True(t, internal.Is(err, ogerr))

	// wrap accepts wrapper args
	err = WrapSkipping(err, 0, WithUserMessage("hi"), WithHTTPCode(6))
	assert.Equal(t, "hi", UserMessage(err))
	assert.Equal(t, 6, HTTPCode(err))

	// wrap should only capture the stack once, even if non-merry errors are in the chain
	// between the top and where the stack is
	err = &UnwrapperError{err}
	err = Wrap(err, WithHTTPCode(55))
	count := 0
	err1 := err
	for err1 != nil {
		if impl, ok := err.(*errImpl); ok {
			if impl.key == errKeyStack {
				count++
			}
		}
		err1 = internal.Unwrap(err1)
	}

	// wrapping nil -> nil
	assert.Nil(t, WrapSkipping(nil, 1))
}

type Valuer struct{}

func (Valuer) Error() string {
	return "boom"
}

func (Valuer) Value(key interface{}) interface{} {
	return "bam"
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Value(nil, "color"))

	err := New("bang")
	assert.Nil(t, Value(err, "color"))

	err = Wrap(err, WithValue("color", "red"))
	assert.Equal(t, "red", Value(err, "color"))

	// supports interface
	err = Wrap(&Valuer{})
	assert.Equal(t, "bam", Value(err, "color"))

	// will traverse non-merry errors in the chain
	err = New("bam", WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = Wrap(err, WithUserMessage("yikes"))
	assert.Equal(t, "red", Value(err, "color"))

	// will not traverse causes
	err = New("whoops", WithCause(err))
	assert.Equal(t, "red", Value(err, "color"))
}

func TestValues(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Values(nil))

	// error with no values should still be nil
	assert.Nil(t, Values(errors.New("boom")))

	// create an error chain with a few values attached, and a non-merry error
	// in the middle.
	err := New("boom", WithUserMessage("bam"), WithHTTPCode(4))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("color", "red"))

	values := Values(err)

	assert.Equal(t, map[interface{}]interface{}{
		errKeyStack: Stack(err),
		errKeyUserMessage: "bam",
		errKeyHTTPCode: 4,
		"color": "red",
	}, values)
}

func TestStack(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Stack(nil))

	// error without a stack
	assert.Nil(t, Stack(errors.New("boom")))
	assert.Nil(t, Stack(New("boom", NoCaptureStack())))

	// error with stack
	assert.NotEmpty(t, Stack(New("boom")))

	// works when value is deep in stack
	err := New("bam", WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = Wrap(err, WithUserMessage("yikes"))
	assert.NotEmpty(t, Stack(err))
}

func TestHTTPCode(t *testing.T) {
	// nil -> 200
	assert.Equal(t, 200, HTTPCode(nil))

	// default to 500
	assert.Equal(t, 500, HTTPCode(errors.New("boom")))

	// set with wrapper
	assert.Equal(t, 404, HTTPCode(New("boom", WithHTTPCode(404))))

	// works when value is deep in stack
	err := New("bam", WithHTTPCode(404))
	err = &UnwrapperError{err}
	err = Wrap(err, WithUserMessage("yikes"))
	assert.Equal(t, 404, HTTPCode(err))
}

func TestUserMessage(t *testing.T) {
	// nil -> empty
	assert.Empty(t, UserMessage(nil))

	// default to empty
	assert.Empty(t, UserMessage(New("boom")))

	// set with wrapper
	assert.Equal(t, "bang", UserMessage(New("boom", WithUserMessage("bang"))))

	// works when value is deep in stack
	err := New("bam", WithUserMessage("red"))
	err = &UnwrapperError{err}
	err = Wrap(err, WithHTTPCode(404))
	assert.Equal(t, "red", UserMessage(err))
}

type Causer struct {
	msg string
	cause error
}

func (c *Causer) Error() string {
	return c.msg
}

func (c *Causer) Cause() error {
	return c.cause
}

func TestCause(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Cause(nil))

	// no cause -> nil
	assert.Nil(t, Cause(New("boom")))

	// with cause
	root := errors.New("boom")
	err := New("yikes", WithCause(root))
	assert.EqualError(t, Cause(err), "boom")

	// works with causer interface
	err = errors.New("boom")
	err = &Causer{msg: "yikes", cause: err}
	err = Wrap(err, WithUserMessage("red"))
	assert.EqualError(t, Cause(err), "boom")

}

func TestHasStack(t *testing.T) {
	// nil -> false
	assert.False(t, HasStack(nil))

	// errors without stacks
	assert.False(t, HasStack(errors.New("boom")))

	// errors with stacks
	assert.True(t, HasStack(New("boom")))
	assert.True(t, HasStack(New("boom", NoCaptureStack())))
}