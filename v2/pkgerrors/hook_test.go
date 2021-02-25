package pkgerrors

import (
	"github.com/ansel1/merry/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestHook(t *testing.T) {
	merry.ClearHooks()
	Install()

	var err error

	_, _, rl, _ := runtime.Caller(0)
	err = errors.New("crash")
	err = merry.Wrap(err, merry.WithMessage("yikes"))

	assert.EqualError(t, err, "yikes")
	file, line := merry.Location(err)

	assert.Contains(t, file, "hook_test.go")
	assert.Equal(t, rl+1, line)
}
