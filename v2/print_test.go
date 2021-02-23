package merry

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestLocation(t *testing.T) {
	// nil -> nil
	f, l := Location(nil)
	assert.Equal(t, "", f)
	assert.Equal(t, 0, l)

	// err with no stack
	f, l = Location(errors.New("hi"))
	assert.Equal(t, "", f)
	assert.Equal(t, 0, l)

	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	f, l = Location(err)
	assert.Contains(t, f, "errors_test.go")
	assert.Equal(t, rl+1, l)
}

func TestSourceLine(t *testing.T) {
	// nil -> empty
	line := SourceLine(nil)
	assert.Empty(t, line)

	// err with no stack
	line = SourceLine(errors.New("hi"))
	assert.Empty(t, line)

	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	line = SourceLine(err)
	assert.Equal(t, fmt.Sprintf("github.com/ansel1/merry/v2.TestSourceLine (print_test.go:%v)",rl + 1), line)
}

func TestFormattedStack(t *testing.T) {
	// nil -> nil
	assert.Nil(t, FormattedStack(nil))

	// no stack attached -> nil
	assert.Nil(t, FormattedStack(errors.New("asdf")))

	// err with stack
	_, _, rl, _ := runtime.Caller(0)
	err := New("bang")
	lines := FormattedStack(err)
	assert.NotEmpty(t, lines)
	assert.Regexp(t, `github\.com/ansel1/merry/v2\.TestFormattedStack\n\t.+print_test.go:` + strconv.Itoa(rl+1), lines[0])

	// formatted stack can be set explicitly
	fakeStack := []string{"blue", "red"}
	err = New("boom", WithFormattedStack(fakeStack))
	assert.Equal(t, fakeStack, FormattedStack(err))
}

func TestStacktrace(t *testing.T) {
	// nil -> empty
	assert.Empty(t, Stacktrace(nil))

	// no stack attached -> empty
	assert.Empty(t, Stacktrace(errors.New("hi")))

	// err with stack
	err := New("bang")
	lines := FormattedStack(err)
	assert.NotEmpty(t, lines)
	assert.Equal(t, strings.Join(lines, "\n"), Stacktrace(err))

	// formatted stack can be set explicitly
	err = New("boom", WithFormattedStack([]string{"blue", "red"}))
	assert.Equal(t, "blue\nred", Stacktrace(err))
}

func TestDetails(t *testing.T) {
	// nil -> empty
	assert.Empty(t, Details(nil))

	err := New("bang", WithUserMessage("stay calm"))
	deets := Details(err)
	t.Log(deets)
	lines := strings.Split(deets, "\n")
	assert.Equal(t, "bang", lines[0])
	assert.Contains(t, deets, Stacktrace(err))
	assert.Contains(t, deets, "User Message: stay calm")
}