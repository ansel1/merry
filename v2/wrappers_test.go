package merry

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestWrappers(t *testing.T) {
	tests := []struct {
		name string
		wrapper Wrapper
		assertions func(*testing.T, error)
	}{
		{
			name:"WithValue",
			wrapper: WithValue("color", "red"),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, "red", Value(err, "color"))
			},
		},
		{
			name: "WithMessage",
			wrapper: WithMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom")
			},
		},
		{
			name:       "WithMessagef",
			wrapper:    WithMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "big boom")
			},
		},
		{
			name:       "WithUserMessage",
			wrapper:    WithUserMessage("boom"),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, "boom", UserMessage(err))
			},
		},
		{
			name:       "WithUserMessagef",
			wrapper:    WithUserMessagef("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, "big boom", UserMessage(err))
			},
		},
		{
			name:       "Append",
			wrapper:    Append("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: boom")
			},
		},
		{
			name:       "Appendf",
			wrapper:    Appendf("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "bang: big boom")
			},
		},
		{
			name:       "Prepend",
			wrapper:    Prepend("boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom: bang")
			},
		},
		{
			name:       "Prependf",
			wrapper:    Prependf("%s %s", "big", "boom"),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, err, "big boom: bang")
			},
		},
		{
			name:       "WithHTTPCode",
			wrapper:    WithHTTPCode(56),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, 56, HTTPCode(err))
			},
		},
		{
			name:       "WithStack",
			wrapper:    WithStack([]uintptr{1, 2, 3, 4, 5}),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, []uintptr{1, 2, 3, 4, 5}, Stack(err))
			},
		},
		{
			name:       "WithFormattedStack",
			wrapper:    WithFormattedStack([]string{"blue", "red"}),
			assertions: func(t *testing.T, err error) {
				assert.Equal(t, []string{"blue", "red"}, FormattedStack(err))
			},
		},
		{
			name:       "NoCaptureStack",
			wrapper:    NoCaptureStack(),
			assertions: func(t *testing.T, err error) {
				assert.Nil(t, Stack(err))
			},
		},
		{
			name:       "CaptureStack",
			wrapper:    CaptureStack(false),
			assertions: func(t *testing.T, err error) {
				assert.NotEmpty(t, Stack(err))
			},
		},
		{
			name:       "WithCause",
			wrapper:    WithCause(errors.New("crash")),
			assertions: func(t *testing.T, err error) {
				assert.EqualError(t, Cause(err), "crash")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// nil -> nil
			assert.Nil(t, test.wrapper.Wrap(nil, 0))

			err := test.wrapper.Wrap(errors.New("bang"), 0)
			test.assertions(t, err)
		})
	}
}

func TestSet(t *testing.T) {
	// nil -> nil
	assert.Nil(t, Set(nil, "color", "red"))

	err := Set(errors.New("bang"), "color", "red")
	assert.Equal(t, "red", Value(err, "color"))
}

func TestNoCaptureStack(t *testing.T) {
	// without the option, a stack should be captured
	err := New("bang")
	assert.NotEmpty(t, Stack(err))

	// with option, stack capture should be suppressed
	err = New("bang", NoCaptureStack())
	assert.Nil(t, Stack(err))

	// even if the err is wrapped some more, capture should be
	// suppressed
	err = Wrap(err)
	assert.Nil(t, Stack(err))

	// should also work when wrapping an external error
	err = Wrap(errors.New("bang"), NoCaptureStack())
	assert.Nil(t, Stack(err))
}

func TestCaptureStack(t *testing.T) {
	defer SetStackCaptureEnabled(true)

	// if stack is already captured, will capture a new
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	_, l := Location(err)
	assert.Equal(t, rl+1, l)

	_, _, rl, _ = runtime.Caller(0)
	err = Wrap(err, CaptureStack(false))
	_, l = Location(err)
	assert.Equal(t, rl+1, l)

	// if global capture disabled, it won't capture a stack
	SetStackCaptureEnabled(false)
	err = New("bang", CaptureStack(false))
	assert.Nil(t, Stack(err))

	// force can be used to override global flag
	_, _, rl, _ = runtime.Caller(0)
	err = Wrap(err, CaptureStack(true))
	_, l = Location(err)
	assert.Equal(t, rl+1, l)
}
