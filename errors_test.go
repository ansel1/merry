package merry

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
)

func TestNew(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	if HTTPCode(err) != 500 {
		t.Errorf("http code should have been 500, was %v", HTTPCode(err))
	}
	if err.Error() != "bang" {
		t.Errorf("error message should have been bang, was %v", err.Error())
	}
	f, l := Location(err)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl+1 {
		t.Errorf("error line should have been %d, was %d", rl+1, 8)
	}
}

func TestErrorf(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := Errorf("chitty chitty %v %v", "bang", "bang")
	if HTTPCode(err) != 500 {
		t.Errorf("http code should have been 500, was %v", HTTPCode(err))
	}
	if err.Error() != "chitty chitty bang bang" {
		t.Errorf("error message should have been chitty chitty bang bang, was %v", err.Error())
	}
	f, l := Location(err)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl+1 {
		t.Errorf("error line should have been %d, was %d", rl+1, 8)
	}
}

func TestUserError(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := UserError("bang")
	assert.Equal(t, "bang", UserMessage(err))
	assert.Empty(t, Message(err))
	_, l := Location(err)
	assert.Equal(t, rl+1, l)
}

func TestUserErrorf(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := UserErrorf("bang %v", "bang")
	assert.Equal(t, "bang bang", UserMessage(err))
	assert.Empty(t, Message(err))
	_, l := Location(err)
	assert.Equal(t, rl+1, l)
}

func TestDetails(t *testing.T) {
	var err error = New("bang")
	deets := Details(err)
	t.Log(deets)
	lines := strings.Split(deets, "\n")
	if lines[0] != "bang" {
		t.Errorf("first line should have been bang: %v", lines[0])
	}
	if !strings.Contains(deets, Stacktrace(err)) {
		t.Error("should have contained the error stacktrace")
	}

	err = WithUserMessage(err, "stay calm")
	deets = Details(err)
	t.Log(deets)
	assert.Contains(t, deets, "User Message: stay calm")

	// Allow nil error
	assert.Empty(t, Details(nil))
}

func TestStacktrace(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	var err error = New("bang")

	assert.NotEmpty(t, Stack(err))
	s := Stacktrace(err)
	t.Log(s)
	lines := strings.Split(s, "\n")
	assert.NotEmpty(t, lines)
	assert.Equal(t, "github.com/ansel1/merry.TestStacktrace", lines[0])
	assert.Contains(t, lines[1], fmt.Sprintf("errors_test.go:%d", rl+1))
	// Allow nil error
	assert.Empty(t, Stacktrace(nil))
}

func TestWrap(t *testing.T) {
	err := errors.New("simple")
	_, _, rl, _ := runtime.Caller(0)
	wrapped := WrapSkipping(err, 0)
	f, l := Location(wrapped)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl+1 {
		t.Errorf("error line should have been %d, was %d", rl+1, l)
	}

	rich2 := WrapSkipping(wrapped, 0)
	if wrapped != rich2 {
		t.Error("rich and rich2 are not the same.  Wrap should have been no-op if rich was already a RichError")
	}
	if !reflect.DeepEqual(Stack(wrapped), Stack(rich2)) {
		t.Log(Details(rich2))
		t.Error("wrap should have left the stacktrace alone if the original error already had a stack")
	}
	// wrapping nil -> nil
	assert.Nil(t, Wrap(nil))
	assert.Nil(t, WrapSkipping(nil, 1))
}

func TestHere(t *testing.T) {
	parseError := New("Parse error")
	invalidCharSet := WithMessage(parseError, "Invalid charset").WithHTTPCode(400)
	invalidSyntax := parseError.WithMessage("Syntax error")

	if !Is(invalidCharSet, parseError) {
		t.Error("invalidCharSet should be a parseError")
	}

	_, _, rl, _ := runtime.Caller(0)
	pe := Here(parseError)
	_, l := Location(pe)
	if l != rl+1 {
		t.Errorf("Here should capture a new stack.  Expected %d, got %d", rl+1, l)
	}

	if !Is(pe, parseError) {
		t.Error("pe should be a parseError")
	}
	if Is(pe, invalidCharSet) {
		t.Error("pe should not be an invalidCharSet")
	}
	if pe.Error() != "Parse error" {
		t.Errorf("child error's message is wrong, expected: Parse error, got %v", pe.Error())
	}
	icse := Here(invalidCharSet)
	if !Is(icse, parseError) {
		t.Error("icse should be a parseError")
	}
	if !Is(icse, invalidCharSet) {
		t.Error("icse should be an invalidCharSet")
	}
	if Is(icse, invalidSyntax) {
		t.Error("icse should not be an invalidSyntax")
	}
	if icse.Error() != "Invalid charset" {
		t.Errorf("child's message is wrong.  Expected: Invalid charset, got: %v", icse.Error())
	}
	if HTTPCode(icse) != 400 {
		t.Errorf("child's http code is wrong.  Expected 400, got %v", HTTPCode(icse))
	}

	// nil -> nil
	assert.Nil(t, Here(nil))
}

func TestHereSkipping(t *testing.T) {
	var e error = New("boom")

	f := func() error {
		return HereSkipping(e, 1)
	}

	_, _, rl, _ := runtime.Caller(0)
	e = f()

	_, l := Location(e)
	require.Equal(t, rl+1, l)
}

func TestUnwrap(t *testing.T) {
	inner := errors.New("bing")
	wrapper := WrapSkipping(inner, 0)
	if Unwrap(wrapper) != inner {
		t.Errorf("unwrapped error should have been the inner err, was %#v", inner)
	}

	doubleWrap := wrapper.WithMessage("blag")
	if Unwrap(doubleWrap) != inner {
		t.Errorf("unwrapped should recurse to inner, but got %#v", inner)
	}

	// nil -> nil
	assert.Nil(t, Unwrap(nil))
}

func TestNilValues(t *testing.T) {
	// Quirk of go
	// http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/index.html#nil_in_nil_in_vals
	// an interface value isn't nil unless both the type *and* the value are nil
	// make sure we aren't accidentally returning nil values but non-nil types
	type e struct{}
	var anE *e
	type f interface{}
	var anF f
	if anF != nil {
		t.Error("anF should have been nil here, because it doesn't have a concete type yet")
	}
	anF = anE
	if anF == nil {
		t.Error("anF should have been not nil here, because it now has a concrete type")
	}
	if WithMessage(WithHTTPCode(Wrap(nil), 400), "hey") != nil {
		t.Error("by using interfaces in all the returns, this should have remained a true nil value")
	}
}

func TestIs(t *testing.T) {
	ParseError := errors.New("blag")
	cp := Here(ParseError)
	if !Is(cp, ParseError) {
		t.Error("Is(child, parent) should be true")
	}
	if Is(ParseError, cp) {
		t.Error("Is(parent, child) should not be true")
	}
	if !Is(ParseError, ParseError) {
		t.Error("errors are always themselves")
	}
	if !Is(cp, cp) {
		t.Error("should work when comparing rich error to itself")
	}
	if Is(Here(ParseError), cp) {
		t.Error("Is(sibling, sibling) should not be true")
	}
	err2 := errors.New("blag")
	if Is(ParseError, err2) {
		t.Error("These should not have been equal")
	}
	if Is(Here(err2), cp) {
		t.Error("these were not copies of the same error")
	}
	if Is(Here(err2), ParseError) {
		t.Error("underlying errors were not equal")
	}

	nilTests := []struct {
		arg1, arg2 error
		expect     bool
		msg        string
	}{
		{nil, New("t"), false, "nil is not any concrete error"},
		{New("t"), nil, false, "no concrete error is nil"},
		{nil, nil, true, "nil is nil"},
	}
	for _, tst := range nilTests {
		assert.Equal(t, tst.expect, Is(tst.arg1, tst.arg2), tst.msg)
	}
}

func TestHTTPCode(t *testing.T) {
	basicErr := errors.New("blag")
	if c := HTTPCode(basicErr); c != 500 {
		t.Errorf("default code should be 500, was %d", c)
	}
	err := New("blug")
	if c := HTTPCode(err); c != 500 {
		t.Errorf("default code for rich errors should be 500, was %d", c)
	}
	errWCode := err.WithHTTPCode(404)
	if c := HTTPCode(errWCode); c != 404 {
		t.Errorf("the code should be set to 404, was %d", c)
	}
	if HTTPCode(err) != 500 {
		t.Error("original error should not have been modified")
	}

	// nil -> nil
	assert.Nil(t, WithHTTPCode(nil, 404))
	assert.Equal(t, 200, HTTPCode(nil), "The code for nil is 200 (ok)")
}

func TestImplicitWrapping(t *testing.T) {
	// WithXXX functions will implicitly wrap non-merry errors
	// but if they do so, they should skip a frame, so the merry error's stack
	// appears to start wherever the WithXXX function was called

	_, _, rl, _ := runtime.Caller(0)
	tests := []struct {
		f     func() error
		fname string
	}{
		{fname: "WithHTTPCode", f: func() error { return WithHTTPCode(errors.New("bug"), 404) }},
		{fname: "WithUserMessage", f: func() error { return WithUserMessage(errors.New("bug"), "asdf") }},
		{fname: "WithUserMessages", f: func() error { return WithUserMessagef(errors.New("bug"), "asdf") }},
		{fname: "WithMessage", f: func() error { return WithMessage(errors.New("bug"), "asdf") }},
		{fname: "WithMessagef", f: func() error { return WithMessagef(errors.New("bug"), "asdf") }},
		{fname: "WithValue", f: func() error { return WithValue(errors.New("bug"), "asdf", "asdf") }},
		{fname: "Append", f: func() error { return Append(errors.New("bug"), "asdf") }},
		{fname: "Appendf", f: func() error { return Appendf(errors.New("bug"), "asdf") }},
		{fname: "Prepend", f: func() error { return Prepend(errors.New("bug"), "asdf") }},
		{fname: "Prependf", f: func() error { return Prependf(errors.New("bug"), "asdf") }},
	}
	for i, test := range tests {
		t.Log("Testing ", test.fname)
		err := test.f()
		f, l := Location(err)
		assert.Contains(t, f, "errors_test.go", "error message should have contained errors_test.go")
		assert.Equal(t, rl+5+i, l, "error line number was incorrect")
	}
}

func TestWithMessage(t *testing.T) {
	err1 := New("blug")
	err2 := err1.WithMessage("blee")
	err3 := err2.WithMessage("red")
	assert.EqualError(t, err1, "blug")
	assert.EqualError(t, err2, "blee", "should have overridden the underlying message")
	assert.EqualError(t, err3, "red")
	assert.Equal(t, Stack(err1), Stack(err2), "stack should not have been altered")

	// nil -> nil
	assert.Nil(t, WithMessage(nil, ""))
}

func TestWithMessagef(t *testing.T) {
	err1 := New("blug")
	err2 := err1.WithMessagef("super %v", "stew")
	err3 := err1.WithMessagef("blue %v", "red")
	assert.EqualError(t, err1, "blug")
	assert.EqualError(t, err2, "super stew")
	assert.EqualError(t, err3, "blue red")
	assert.Equal(t, Stack(err1), Stack(err2), "stack should not have been altered")
	// nil -> nil
	assert.Nil(t, WithMessagef(nil, "", ""))
}

func TestMessage(t *testing.T) {
	tests := []error{
		errors.New("one"),
		WithMessage(errors.New("blue"), "one"),
		New("one"),
	}
	for _, test := range tests {
		assert.Equal(t, "one", test.Error())
		assert.Equal(t, "one", Message(test))
	}

	// when verbose is on, Error() changes, but Message() doesn't
	defer SetVerboseDefault(false)
	SetVerboseDefault(true)
	e := New("two")
	assert.Equal(t, "two", Message(e))
	assert.NotEqual(t, "two", e.Error())

	// when error is nil, return ""
	assert.Empty(t, Message(nil))

}

func TestWithUserMessage(t *testing.T) {
	fault := New("seg fault")
	e := WithUserMessage(fault, "a glitch")
	assert.Equal(t, "seg fault", e.Error())
	assert.Equal(t, "a glitch", UserMessage(e))
	e = WithUserMessagef(e, "not a %s deal", "huge")
	assert.Equal(t, "not a huge deal", UserMessage(e))
	// If user message is set and regular message isn't, set regular message to user message
	e = New("").WithUserMessage("a blag")
	assert.Equal(t, "a blag", UserMessage(e))
	assert.Equal(t, "a blag", e.Error())
}

func TestAppend(t *testing.T) {
	blug := New("blug")
	err := blug.Append("blog")
	assert.Equal(t, err.Error(), "blug: blog")
	err = Append(err, "blig")
	assert.Equal(t, err.Error(), "blug: blog: blig")
	err = blug.Appendf("%s", "blog")
	assert.Equal(t, err.Error(), "blug: blog")
	err = Appendf(err, "%s", "blig")
	assert.Equal(t, err.Error(), "blug: blog: blig")

	// nil -> nil
	assert.Nil(t, Append(nil, ""))
	assert.Nil(t, Appendf(nil, "", ""))
}

func TestPrepend(t *testing.T) {
	blug := New("blug")
	err := blug.Prepend("blog")
	assert.Equal(t, err.Error(), "blog: blug")
	err = Prepend(err, "blig")
	assert.Equal(t, err.Error(), "blig: blog: blug")
	err = blug.Prependf("%s", "blog")
	assert.Equal(t, err.Error(), "blog: blug")
	err = Prependf(err, "%s", "blig")
	assert.Equal(t, err.Error(), "blig: blog: blug")

	// nil -> nil
	assert.Nil(t, Prepend(nil, ""))
	assert.Nil(t, Prependf(nil, "", ""))
}

func TestLocation(t *testing.T) {
	// nil -> nil
	f, l := Location(nil)
	assert.Equal(t, "", f)
	assert.Equal(t, 0, l)
}

func TestSourceLine(t *testing.T) {
	source := SourceLine(nil)
	assert.Equal(t, source, "")

	var err error = New("foo")
	source = SourceLine(err)
	t.Log(source)
	assert.NotEqual(t, source, "")

	p := regexp.MustCompile(`^.*errors_test\.go:(\d+)$`)

	parts := p.FindStringSubmatch(source)
	require.NotNil(t, parts, "source did not match path pattern: %v", source)

	if i, e := strconv.Atoi(parts[1]); e != nil {
		t.Errorf("not a number: %s", parts[1])
	} else if i <= 0 {
		t.Errorf("source line must be > 1: %s", parts[1])
	}
}

func TestValue(t *testing.T) {
	// nil -> nil
	assert.Nil(t, WithValue(nil, "", ""))
	assert.Nil(t, Value(nil, ""))
}

func TestValues(t *testing.T) {
	// nil -> nil
	values := Values(nil)
	assert.Nil(t, values)

	var e error
	e = New("bad stuff")
	e = WithValue(e, "key1", "val1")
	e = WithValue(e, "key2", "val2")

	values = Values(e)
	assert.NotNil(t, values)
	assert.Equal(t, values["key1"], "val1")
	assert.Equal(t, values["key2"], "val2")
	assert.NotNil(t, values[stack])

	// make sure the last value attached is returned
	e = WithValue(e, "key3", "val3")
	e = WithValue(e, "key3", "val4")
	values = Values(e)
	assert.Equal(t, values["key3"], "val4")

}

func TestStackCaptureEnabled(t *testing.T) {
	// on by default
	assert.True(t, StackCaptureEnabled())

	SetStackCaptureEnabled(false)
	assert.False(t, StackCaptureEnabled())
	e := New("yikes")
	assert.Empty(t, Stack(e))
	// let's just make sure none of the print functions bomb when there's no stack
	assert.Empty(t, SourceLine(e))
	f, l := Location(e)
	assert.Empty(t, f)
	assert.Equal(t, 0, l)
	assert.Empty(t, Stacktrace(e))
	assert.NotPanics(t, func() { Details(e) })

	// turn it back on
	SetStackCaptureEnabled(true)
	assert.True(t, StackCaptureEnabled())

	e = New("mommy")
	assert.NotEmpty(t, Stack(e))
}

func TestVerboseDefault(t *testing.T) {
	defer SetVerboseDefault(false)
	// off by default
	assert.False(t, VerboseDefault())

	SetVerboseDefault(true)
	assert.True(t, VerboseDefault())
	e := New("yikes")
	// test verbose on
	assert.Equal(t, Details(e), e.Error())
	// test verbose off
	SetVerboseDefault(false)
	s := e.Error()
	assert.Equal(t, Message(e), s)
	assert.Equal(t, "yikes", s)
}

func TestMerryErr_Error(t *testing.T) {
	origVerbose := verbose
	defer func() {
		verbose = origVerbose
	}()

	// test with verbose on
	verbose = false

	tests := []struct {
		desc                 string
		verbose              bool
		message, userMessage string
		expected             string
	}{
		{
			desc:     "with message",
			message:  "blue",
			expected: "blue",
		},
		{
			desc:        "with user message",
			userMessage: "red",
			expected:    "red",
		},
	}
	for _, test := range tests {
		t.Log("error message tests: " + test.desc)
		verbose = test.verbose
		err := New(test.message).WithUserMessage(test.userMessage)
		t.Log(err.Error())
		assert.Equal(t, test.expected, err.Error())
	}

}

func TestMerryErr_Format(t *testing.T) {
	e := New("Hi")
	assert.Equal(t, fmt.Sprintf("%v", e), e.Error())
	assert.Equal(t, fmt.Sprintf("%s", e), e.Error())
	assert.Equal(t, fmt.Sprintf("%q", e), fmt.Sprintf("%q", e.Error()))
	assert.Equal(t, fmt.Sprintf("%+v", e), Details(e))
}

func TestCause(t *testing.T) {

	e1 := New("low level error")
	e2 := New("high level error")

	e3 := WithCause(e2, e1)
	e4 := New("top level error")
	e5 := e4.WithCause(e3)

	assert.True(t, Is(e3, e1))
	assert.True(t, Is(e3, e2))

	assert.Equal(t, e1, Cause(e3))
	assert.Equal(t, e1, e3.(Error).Cause())

	assert.Nil(t, Cause(e2))
	assert.Nil(t, Cause(e1))
	assert.Nil(t, e1.(Error).Cause())
	assert.Nil(t, e2.(Error).Cause())

	assert.Equal(t, e3.Error(), e2.Error()+": "+e1.Error())

	assert.True(t, Is(e5, e4))
	assert.True(t, Is(e5, e3))
	assert.True(t, Is(e5, e2))
	assert.True(t, Is(e5, e1))

	assert.Equal(t, e3, Cause(e5))
	assert.NotEqual(t, e3, RootCause(e5))
	assert.Equal(t, e1, RootCause(e5))

	// ensure cause message isn't double appended
	assert.Equal(t, "red: high level error: low level error", Prepend(e3, "red").Error())
}

func BenchmarkNew_withStackCapture(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New("boom")
	}
}

func BenchmarkNew_withoutStackCapture(b *testing.B) {
	SetStackCaptureEnabled(false)
	for i := 0; i < b.N; i++ {
		New("boom")
	}
}
