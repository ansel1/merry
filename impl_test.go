package merry

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type Wrapper struct {
	err error
}

func (w *Wrapper) Error() string {
	return w.err.Error()
}

func (w *Wrapper) Unwrap() error {
	return w.err
}

func TestErrImpl_Unwrap(t *testing.T) {
	e1 := New("blue")
	c1 := New("fm")
	e2 := Prepend(e1, "color")
	e3 := e2.WithCause(c1)

	assert.Equal(t, e2, unwrap(e3))
	assert.Equal(t, e1, unwrap(e2))
}

func TestErrImpl_Is(t *testing.T) {
	e1 := New("blue")
	c1 := New("fm")
	e2 := Prepend(e1, "color").WithCause(c1)
	e3 := New("red")

	assert.True(t, is(e2, e2))
	assert.True(t, is(e2, e1))
	assert.True(t, is(e2, c1))
	assert.False(t, is(e2, e3))

	// test that it works with non-merry errors in the chain
	var e4 error = &Wrapper{e2}
	var e5 error = Prepend(e4, "asdf")

	assert.True(t, is(e5, e2))
	assert.True(t, is(e4, e2))
	assert.True(t, is(e5, c1))
	assert.True(t, is(e5, e1))
}

type redError int

func (*redError) Error() string {
	return "red error"
}

func TestErrImpl_As(t *testing.T) {
	e1 := New("blue error")

	var rerr *redError
	assert.False(t, as(e1, &rerr))
	assert.Nil(t, rerr)

	rr := redError(3)
	e2 := Wrap(&rr)

	assert.True(t, as(e2, &rerr))
	assert.Equal(t, &rr, rerr)

	// test that it works with non-merry errors in the chain
	w := &Wrapper{err: e2}
	e3 := Prepend(w, "asdf")

	rerr = nil

	assert.True(t, as(e3, &rerr))
	assert.Equal(t, &rr, rerr)

	rerr = nil

	assert.True(t, as(w, &rerr))
	assert.Equal(t, &rr, rerr)
}

func BenchmarkIs(b *testing.B) {
	root := New("root")
	err := root
	for i := 0; i < 1000; i++ {
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
