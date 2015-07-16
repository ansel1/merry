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

func TestDetails(t *testing.T) {
	var err error = New("bang")
	deets := Details(err)
	t.Log(deets)
	lines := strings.Split(deets, "\n")
	if lines[0] != "bang" {
		t.Errorf("first line should have been bang", lines[0])
	}
	if !strings.Contains(deets, Stacktrace(err)) {
		t.Errorf("should have contained the error stacktrace")
	}
	// Allow nil error
	assert.Empty(t, Details(nil))
}

func TestStacktrace(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	var err error = New("bang")
	if !(len(Stack(err)) > 0) {
		t.Fatalf("stack length is 0")
	}
	s := Stacktrace(err)
	t.Log(s)
	lines := strings.Split(s, "\n")
	if len(lines) < 1 {
		t.Fatalf("stacktrace is empty")
	}
	if !strings.Contains(lines[0], fmt.Sprintf("errors_test.go:%d", rl+1)) {
		t.Fatalf("stacktrace is wrong")
	}
	// Allow nil error
	assert.Empty(t, Stacktrace(nil))
}

func TestWrap(t *testing.T) {
	var err error = errors.New("simple")
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
	ParseError := New("Parse error")
	InvalidCharSet := WithMessage(ParseError, "Invalid charset").WithHTTPCode(400)
	InvalidSyntax := ParseError.WithMessage("Syntax error")

	if !Is(InvalidCharSet, ParseError) {
		t.Error("InvalidCharSet should be a ParseError")
	}

	_, _, rl, _ := runtime.Caller(0)
	pe := Here(ParseError)
	_, l := Location(pe)
	if l != rl+1 {
		t.Errorf("Extend should capture a new stack.  Expected %d, got %d", rl+1, l)
	}

	if !Is(pe, ParseError) {
		t.Error("pe should be a ParseError")
	}
	if Is(pe, InvalidCharSet) {
		t.Error("pe should not be an InvalidCharSet")
	}
	if pe.Error() != "Parse error" {
		t.Errorf("child error's message is wrong, expected: Parse error, got %v", pe.Error())
	}
	icse := Here(InvalidCharSet)
	if !Is(icse, ParseError) {
		t.Error("icse should be a ParseError")
	}
	if !Is(icse, InvalidCharSet) {
		t.Error("icse should be an InvalidCharSet")
	}
	if Is(icse, InvalidSyntax) {
		t.Error("icse should not be an InvalidSyntax")
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
	copy := Here(ParseError)
	if !Is(copy, ParseError) {
		t.Error("Is(child, parent) should be true")
	}
	if Is(ParseError, copy) {
		t.Error("Is(parent, child) should not be true")
	}
	if !Is(ParseError, ParseError) {
		t.Error("errors are always themselves")
	}
	if !Is(copy, copy) {
		t.Error("should work when comparing rich error to itself")
	}
	if Is(Here(ParseError), copy) {
		t.Error("Is(sibling, sibling) should not be true")
	}
	err2 := errors.New("blag")
	if Is(ParseError, err2) {
		t.Error("These should not have been equal")
	}
	if Is(Here(err2), copy) {
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

func TestOverridingMessage(t *testing.T) {
	blug := New("blug")
	err := blug.WithMessage("blee")
	if m := err.Error(); m != "blee" {
		t.Errorf("should have overridden the underlying message, expecting blee, was %s", m)
	}
	if m := err.WithMessagef("super %v", "stew").Error(); m != "super stew" {
		t.Errorf("formatted message didn't work.  got %v", m)
	}
	if !reflect.DeepEqual(Stack(blug), Stack(err)) {
		t.Error("err should have the same stack as blug")
	}

	// nil -> nil
	assert.Nil(t, WithMessage(nil, ""))
	assert.Nil(t, WithMessagef(nil, "", ""))
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

func TestLocation(t *testing.T) {
	// nil -> nil
	f, l := Location(nil)
	assert.Equal(t, "", f)
	assert.Equal(t, 0, l)
}

func TestSourceLine(t *testing.T) {
	source := SourceLine(nil)
	assert.Equal(t, source, "")

	err := New("foo")
	source = SourceLine(err)
	assert.NotEqual(t, source, "")

	parts := strings.Split(source, ":")
	assert.Equal(t, len(parts), 2)

	if !strings.HasSuffix(parts[0], "errors_test.go") {
		t.Errorf("source line should contain file name")
	}
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
