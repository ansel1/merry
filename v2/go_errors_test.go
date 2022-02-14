package merry

import (
	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestGoErrorsPreserveStack(t *testing.T) {
	var err error

	_, _, rl, _ := runtime.Caller(0)
	err = errors.New("crash")
	err = Wrap(err, WithMessage("yikes"))

	assert.EqualError(t, err, "yikes")
	file, line := Location(err)

	assert.Contains(t, file, "go_errors_test.go")
	assert.Equal(t, rl+1, line)
}
