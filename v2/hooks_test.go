package merry

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHooks(t *testing.T) {
	ClearHooks()

	var appliedCount int
	hook := WrapperFunc(func(err error, i int) error {
		require.NotNil(t, err)
		appliedCount++
		return err
	})
	AddHooks(hook)

	err := Wrap(errors.New("boom"))
	assert.Equal(t, 1, appliedCount)

	Wrap(err)
	assert.Equal(t, 2, appliedCount)

	ClearHooks()
	appliedCount = 0

	Wrap(errors.New("boom"))
	assert.Zero(t, appliedCount)

	AddOnceHooks(hook)
	err = Wrap(errors.New("boom"))
	assert.Equal(t, 1, appliedCount)
	Wrap(errors.New("boom"))
	assert.Equal(t, 2, appliedCount)
	Wrap(err)
	assert.Equal(t, 2, appliedCount)
}
