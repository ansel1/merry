package richerrors
import (
	"testing"
	"strings"
	"errors"
	"runtime"
	"fmt"
	"reflect"
)

func TestNew(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	if err.HTTPCode() != 500 {
		t.Errorf("http code should have been 500, was %v", err.HTTPCode())
	}
	if err.Error() != "bang" {
		t.Errorf("error message should have been bang, was %v", err.Error())
	}
	f, l := Location(err)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl + 1 {
		t.Errorf("error line should have been %d, was %d", rl + 1, 8)
	}
}

func TestErrorf(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := Errorf("chitty chitty %v %v", "bang", "bang")
	if err.HTTPCode() != 500 {
		t.Errorf("http code should have been 500, was %v", err.HTTPCode())
	}
	if err.Error() != "chitty chitty bang bang" {
		t.Errorf("error message should have been chitty chitty bang bang, was %v", err.Error())
	}
	f, l := Location(err)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl + 1 {
		t.Errorf("error line should have been %d, was %d", rl + 1, 8)
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
}

func TestStacktrace(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	var err error = New("bang")
	st, ok := err.(Stacker)
	if !ok {
		t.Fatalf("err doesn't implement Stacker")
	}
	if !(len(st.Stack()) > 0) {
		t.Fatalf("stack length is 0")
	}
	s := Stacktrace(err)
	t.Log(s)
	lines := strings.Split(s, "\n")
	if len(lines) < 1 {
		t.Fatalf("stacktrace is empty")
	}
	if !strings.Contains(lines[0], fmt.Sprintf("errors_test.go:%d", rl + 1)) {
		t.Fatalf("stacktrace is wrong")
	}
}

func TestWrap(t *testing.T) {
	var err error = errors.New("simple")
	_, _, rl, _ := runtime.Caller(0)
	var rich RichError = Wrap(err, 0)
	f, l := Location(rich)
	if !strings.Contains(f, "errors_test.go") {
		t.Errorf("error message should have contained errors_test.go, was %s", f)
	}
	if l != rl + 1 {
		t.Errorf("error line should have been %d, was %d", rl + 1, l)
	}

	rich2 := Wrap(rich, 0)
	if rich != rich2 {
		t.Error("rich and rich2 are not the same.  Wrap should have been no-op if rich was already a RichError")
	}
	if !reflect.DeepEqual(rich.Stack(), rich2.Stack()) {
		t.Log(Details(rich2))
		t.Error("wrap should have left the stacktrace alone if the original error already had a stack")
	}
}

func TestCopy(t *testing.T) {
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang").WithHTTPCode(404).WithMessage("thppt")
	_, _, rl2, _ := runtime.Caller(0)
	errcopy := Copy(err)
	if errcopy.(*richError).err != err.(*richError).err {
		t.Errorf("The copy's underlying error should be the same as the original.")
	}
	_, l := Location(err)
	if l != rl + 1 {
		t.Errorf("The original's line number is wrong.  Expected %d, got %d", rl + 1, l)
	}
	_, l = Location(errcopy)
	if l != rl2 + 1 {
		t.Errorf("The copy's line number should be %d, got %d", rl2 + 1, l)
	}
	if errcopy.Error() != err.Error() {
		t.Errorf("The copy's error string should be %s, got %s", err.Error(), errcopy.Error())
	}
	if errcopy.HTTPCode() != err.HTTPCode() {
		t.Errorf("The copy's http code should be %d, got %d", err.HTTPCode(), errcopy.HTTPCode())
	}

	serr := errors.New("simple err")
	errcopy = Copy(serr)
	if errcopy.(*richError).err != serr {
		t.Errorf("Copy should work just like wrap if the original error is just a simple error")
	}
}

func TestUnwrap(t *testing.T) {
	inner := errors.New("bing")
	wrapper := Wrap(inner, 0)
	if Unwrap(wrapper) != inner {
		t.Errorf("unwrapped error should have been the inner err, was %#v", inner)
	}

	doubleWrap := New("blag")
	doubleWrap.(*richError).err = wrapper
	if Unwrap(doubleWrap) != inner {
		t.Errorf("unwrapped should recurse to inner, but got %#v", inner)
	}
}

func TestIs(t *testing.T) {
	err := errors.New("blag")
	copy := Copy(err)
	if !Is(err, copy) {
		t.Error("should have compared the wrapped err")
	}
	if !Is(copy, err) {
		t.Error("order of the args shouldn't matter")
	}
	if !Is(err, err) {
		t.Error("should work even with non-richerrors")
	}
	if !Is(copy, copy) {
		t.Error("should work when comparing rich error to itself")
	}
	err2 := errors.New("blag")
	if Is(err, err2) {
		t.Error("These should not have been equal")
	}
	if Is(Copy(err2), copy) {
		t.Error("these were not copies of the same error")
	}
	if Is(Copy(err2), err) {
		t.Error("underlying errors were not equal")
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
	err.WithHTTPCode(404)
	if c := HTTPCode(err); c != 404 {
		t.Errorf("the code should be set to 404, was %d", c)
	}
}

func TestOverridingMessage(t *testing.T) {
	err := New("blug").WithMessage("blee")
	if m := err.Error(); m != "blee" {
		t.Errorf("should have overridden the underlying message, expecting blee, was %s", m)
	}
	err.WithMessage("")
	if m := err.Error(); m != "blug" {
		t.Errorf("should have cleared the wrapper message. expecting blug, was %s", m)
	}
	if m := err.WithMessagef("super %v", "stew").Error(); m != "super stew" {
		t.Errorf("formatted message didn't work.  got %v", m)
	}
}