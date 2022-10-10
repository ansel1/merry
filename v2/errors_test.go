package merry

import (
	"errors"
	"fmt"
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

	// Printing errors wrapped by fmt.Print should include stacktrace (https://github.com/ansel1/merry/issues/26)
	s := fmt.Sprintf("%+v", Errorf("boom: %w", New("bang")))
	assert.Contains(t, s, "errors_test.go")
}

func TestSentinel(t *testing.T) {
	err := Sentinel("boom", WithHTTPCode(5), WrapperFunc(func(err error, depth int) error {
		assert.Equal(t, 3, depth)
		return err
	}))
	assert.EqualError(t, err, "boom")

	assertSentinel(t, err)
}

func TestSentinelf(t *testing.T) {
	err := Sentinelf("%s %s boom", "big", WithHTTPCode(5), "blue", WrapperFunc(func(err error, depth int) error {
		assert.Equal(t, 3, depth)
		return err
	}))
	assert.EqualError(t, err, "big blue boom")

	assertSentinel(t, err)
}

func TestApply(t *testing.T) {
	err := Apply(errors.New("boom"), WithHTTPCode(5), WrapperFunc(func(err error, depth int) error {
		assert.Equal(t, 3, depth)
		return err
	}))
	assert.EqualError(t, err, "boom")

	assertSentinel(t, err)
}

func TestApplySkipping(t *testing.T) {
	err := ApplySkipping(errors.New("boom"), 3, WithHTTPCode(5), WrapperFunc(func(err error, depth int) error {
		assert.Equal(t, 5, depth)
		return err
	}))
	assert.EqualError(t, err, "boom")

	assertSentinel(t, err)
}

func assertSentinel(t *testing.T, err error) {
	t.Helper()

	assert.Equal(t, 5, HTTPCode(err))
	assert.Nil(t, Stack(err))

	_, _, rl, _ := runtime.Caller(0)
	err = Wrap(err)
	_, l := Location(err)
	assert.Equal(t, rl+1, l)
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
	assert.True(t, errors.Is(err, ogerr))

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
		if impl, ok := err.(*errWithValue); ok {
			if impl.key == errKeyStack {
				count++
			}
		}
		err1 = errors.Unwrap(err1)
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
	assert.True(t, errors.Is(err, ogerr))

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
		if impl, ok := err.(*errWithValue); ok {
			if impl.key == errKeyStack {
				count++
			}
		}
		err1 = errors.Unwrap(err1)
	}

	// wrapping nil -> nil
	assert.Nil(t, WrapSkipping(nil, 1))

	// Printing errors wrapped by fmt.Print should include stacktrace (https://github.com/ansel1/merry/issues/26)
	wrappedFmtErr := WrapSkipping(fmt.Errorf("boom: %w", New("bang")), 0)
	s := fmt.Sprintf("%+v", wrappedFmtErr)
	assert.Contains(t, s, "errors_test.go")
	// The wrapper used should not add extraneous nil=nil values to Values()
	values := Values(wrappedFmtErr)
	assert.NotContains(t, values, nil)
}

func TestAppend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Append(nil, "big"))

	// append message
	assert.EqualError(t, Append(New("blue"), "big"), "blue: big")

	// wrapper varargs
	err := Append(New("blue"), "big", WithHTTPCode(3))
	assert.Equal(t, 3, HTTPCode(err))
	assert.EqualError(t, err, "blue: big")
}

func TestAppendf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Appendf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, Appendf(New("blue"), "big %s", "red"), "blue: big red")

	// wrapper varargs
	err := Appendf(New("blue"), "big %s", WithHTTPCode(3), "red")
	assert.Equal(t, 3, HTTPCode(err))
	assert.EqualError(t, err, "blue: big red")
}

func TestPrepend(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Prepend(nil, "big"))

	// append message
	assert.EqualError(t, Prepend(New("blue"), "big"), "big: blue")

	// wrapper varargs
	err := Prepend(New("blue"), "big", WithHTTPCode(3))
	assert.Equal(t, 3, HTTPCode(err))
	assert.EqualError(t, err, "big: blue")
}

func TestPrependf(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Prependf(nil, "big %s", "red"))

	// append message
	assert.EqualError(t, Prependf(New("blue"), "big %s", "red"), "big red: blue")

	// wrapper varargs
	err := Prependf(New("blue"), "big %s", WithHTTPCode(3), "red")
	assert.Equal(t, 3, HTTPCode(err))
	assert.EqualError(t, err, "big red: blue")
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Value(nil, "color"))

	err := New("bang")
	assert.Nil(t, Value(err, "color"))

	err = Wrap(err, WithValue("color", "red"))
	assert.Equal(t, "red", Value(err, "color"))

	// will traverse non-merry errors in the chain
	err = New("bam", WithValue("color", "red"))
	err = &UnwrapperError{err}
	err = Wrap(err, WithUserMessage("yikes"))
	assert.Equal(t, "red", Value(err, "color"))

	// will not search the cause chain
	err = New("whoops", WithCause(New("yikes", WithValue("color", "red"))))
	assert.Nil(t, Value(err, "color"))

	// if the current error and the cause both have a value for the
	// same key, the top errors value will always take precedence, even
	// if the cause was attached to the error after the value was.

	err = New("boom", WithValue("color", "red"))
	err = Wrap(err, WithCause(New("io error", WithValue("color", "blue"))))
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
		errKeyStack:       Stack(err),
		errKeyUserMessage: "bam",
		errKeyHTTPCode:    4,
		"color":           "red",
	}, values)
}

func BenchmarkValues(b *testing.B) {
	// create an error chain with a few values attached, and a non-merry error
	// in the middle.
	err := New("boom", WithUserMessage("bam"), WithHTTPCode(4))
	err = &UnwrapperError{err}
	err = Wrap(err, WithValue("color", "red"))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		Values(err)
	}
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

func TestCause(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Cause(nil))

	// no cause -> nil
	assert.Nil(t, Cause(New("boom")))

	// with cause
	root := errors.New("boom")
	err := New("yikes", WithCause(root))
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

func TestRegisteredDetails(t *testing.T) {
	// nil -> nil
	assert.Nil(t, RegisteredDetails(nil))

	assert.Equal(t, dict{"User Message": nil, "HTTP Code": nil}, RegisteredDetails(New("boom")))
	assert.Equal(t, dict{"User Message": "blue", "HTTP Code": 5}, RegisteredDetails(New("boom", WithUserMessage("blue"), WithHTTPCode(5))))
}

type dict = map[string]interface{}
