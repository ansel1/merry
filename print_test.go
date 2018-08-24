package merry

import (
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestRegisterDetail(t *testing.T) {

	err := New("boom")
	assert.Regexp(t, regexp.MustCompile(`^boom\n\n.*`), Details(err))

	err = err.WithUserMessage("bam")
	assert.Regexp(t, regexp.MustCompile(`^boom\nUser Message: bam\n\n.*`), Details(err))

	err = err.WithHTTPCode(404)
	assert.Regexp(t, regexp.MustCompile(`^boom\nHTTP Code: 404\nUser Message: bam\n\n.*`), Details(err))

	type colorKey int

	err = err.WithValue(colorKey(8), "red")
	// shouldn't appear yet because it hasn't been registered
	assert.Regexp(t, regexp.MustCompile(`^boom\nHTTP Code: 404\nUser Message: bam\n\n.*`), Details(err))

	RegisterDetail("Color", colorKey(8))
	assert.Regexp(t, regexp.MustCompile(`^boom\nColor: red\nHTTP Code: 404\nUser Message: bam\n\n.*`), Details(err))
}
