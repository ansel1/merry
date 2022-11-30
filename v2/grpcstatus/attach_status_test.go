package status

import (
	"errors"
	"github.com/ansel1/merry/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"testing"
)

func TestWithStatus(t *testing.T) {
	// nil -> nil
	assert.Nil(t, AttachStatus(nil, New(codes.Canceled, "blue")))
	assert.EqualError(t, AttachStatus(errors.New("blue"), nil), "blue")

	// attach status
	s := New(codes.Canceled, "red")
	err := AttachStatus(merry.New("blue"), s)
	s1, ok := FromError(err)
	require.True(t, ok)
	s1.Proto()
	assert.Equal(t, s, s1)

	// attach a new status, overrides old
	s2 := New(codes.DeadlineExceeded, "yellow")
	err = AttachStatus(err, s2)
	s3, ok := FromError(err)
	require.True(t, ok)
	s3.Proto()
	assert.Equal(t, s2, s3)
	assert.NotEqual(t, s, s3)
}
