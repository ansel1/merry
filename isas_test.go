package merry

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerryErr_Unwrap(t *testing.T) {
	e1 := New("blue")
	c1 := New("fm")
	e2 := Prepend(e1, "color").WithCause(c1)

	assert.Equal(t, c1, e2.(*merryErr).Unwrap())
}

func TestMerryErr_Is(t *testing.T) {
	e1 := New("blue")
	c1 := New("fm")
	e2 := Prepend(e1, "color").WithCause(c1)
	e3 := New("red")

	assert.True(t, is(e2, e2))
	assert.True(t, is(e2, e1))
	assert.True(t, is(e2, c1))
	assert.False(t, is(e2, e3))
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestMerryErr_As(t *testing.T) {
	e1 := New("blue error")

	var rerr *redError
	assert.False(t, as(e1, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	e2 := Wrap(&rr)

	assert.True(t, as(e2, &rerr))
	assert.Equal(t, &rr, rerr)
}

func BenchmarkIs(b *testing.B) {
	root := New("root")
	err := root
	for i := 0; i < 10; i++ {
		err = New("wrapper").WithCause(err)
		for j := 0; j < 10; j++ {
			err = Prepend(err, "wrapped")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assert.True(b, Is(err, root))
	}
}
