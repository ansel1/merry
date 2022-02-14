package status

import (
	"context"
	"errors"
	"github.com/ansel1/merry/v2"
	"github.com/ansel1/vespucci/v4/mapstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"net/http"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {
	// should passthrough to status package
	s := New(codes.Canceled, "blue")
	s1 := status.New(codes.Canceled, "blue")

	assert.Equal(t, s1, s)
}

func TestNewf(t *testing.T) {
	// should passthrough to status package
	s := Newf(codes.Canceled, "%s blue", "big")
	s1 := status.Newf(codes.Canceled, "%s blue", "big")

	assert.Equal(t, s1, s)
}

func TestError(t *testing.T) {
	// should have a stack
	_, _, rl, _ := runtime.Caller(0)
	err := Error(codes.Canceled, "blue")
	err1 := status.Error(codes.Canceled, "blue")
	assert.EqualError(t, err, err1.Error())

	s1, ok := status.FromError(err1)
	require.True(t, ok)

	s, ok := FromError(err)
	assert.True(t, ok)
	assert.Equal(t, s1, s)

	_, line := merry.Location(err)
	assert.Equal(t, rl+1, line)
}

func TestErrorf(t *testing.T) {
	// should have a stack, but otherwise the same as status package
	_, _, rl, _ := runtime.Caller(0)
	err := Errorf(codes.Canceled, "%s blue", "big")
	err1 := status.Errorf(codes.Canceled, "%s blue", "big")
	assert.EqualError(t, err, err1.Error())

	s1, ok := status.FromError(err1)
	require.True(t, ok)

	s, ok := FromError(err)
	assert.True(t, ok)
	assert.Equal(t, s1, s)

	_, line := merry.Location(err)
	assert.Equal(t, rl+1, line)
}

func TestErrorProto(t *testing.T) {
	s := New(codes.Canceled, "blue")

	// should have a stack, but otherwise the same as status package
	_, _, rl, _ := runtime.Caller(0)
	err := ErrorProto(s.Proto())
	err1 := status.ErrorProto(s.Proto())
	assert.EqualError(t, err, err1.Error())

	s1, ok := FromError(err)
	assert.True(t, ok)
	s1.Proto() // need to call this to set some internal state that's makes the two status' comparable
	assert.Equal(t, s, s1)

	_, line := merry.Location(err)
	assert.Equal(t, rl+1, line)
}

func TestFromProto(t *testing.T) {
	// passthrough to status package
	s := status.New(codes.Canceled, "blue")

	s1 := FromProto(s.Proto())
	s2 := status.FromProto(s.Proto())

	assert.Equal(t, s2, s1)
}

func TestFromError(t *testing.T) {
	// nil -> nil
	s, ok := FromError(nil)
	s1, ok1 := status.FromError(nil)
	assert.Equal(t, ok1, ok)
	assert.Equal(t, s1, s)

	// if err already has a status, return that
	s = New(codes.Canceled, "blue")
	err := s.Err()
	s1, ok = FromError(err)
	s1.Proto()
	assert.Equal(t, s, s1)
	assert.True(t, ok)

	// will also return a status if one of the causes has one
	err = merry.New("one", merry.WithCause(merry.New("two", merry.WithCause(err))))
	s1, ok = FromError(err)
	s1.Proto()
	assert.Equal(t, s, s1)
	assert.True(t, ok)

	// if error has no status already, construct one
	err = merry.New("blue",
		merry.WithHTTPCode(http.StatusUnauthorized),
		merry.WithUserMessage("hi"),
	)

	s, ok = FromError(err)
	assert.False(t, ok)
	assert.Equal(t, "blue", s.Message())
	assert.Equal(t, codes.Unauthenticated, s.Code())
}

func TestConvert(t *testing.T) {
	// just calls FromError
	s := Convert(Error(codes.Canceled, "blue"))
	assert.Equal(t, "blue", s.Message())
	assert.Equal(t, codes.Canceled, s.Code())
}

func TestFromContextError(t *testing.T) {
	// just calls FromError
	s := FromContextError(Error(codes.Canceled, "blue"))
	assert.Equal(t, "blue", s.Message())
	assert.Equal(t, codes.Canceled, s.Code())
}

func TestWithCode(t *testing.T) {
	// nil -> nil
	assert.Nil(t, WithCode(codes.Canceled).Wrap(nil, 0))

	err := merry.New("blue", WithCode(codes.Canceled))
	assert.Equal(t, codes.Canceled, Code(err))

	// WithCode works by cloning any prior attached Status, and changing
	// its code.  Make sure we preserve the rest of the Status.
	s := New(codes.Canceled, "blue")
	s, err = s.WithDetails(&errdetails.LocalizedMessage{Message: "yikes"})
	require.NoError(t, err)

	err = merry.Wrap(s.Err(), WithCode(codes.DeadlineExceeded))
	assert.Equal(t, codes.DeadlineExceeded, Code(err))
	assert.Equal(t, "blue", Convert(err).Message())
	assert.Equal(t, codes.DeadlineExceeded, Convert(err).Code())
	mapstest.AssertContains(t, Convert(err).Details(), &errdetails.LocalizedMessage{Message: "yikes"})
}

func TestCode(t *testing.T) {
	// nil -> ok
	assert.Equal(t, codes.OK, Code(nil))

	// statuser returns status' code
	assert.Equal(t, codes.Canceled, Code(Error(codes.Canceled, "blue")))

	// deadline exceeded
	assert.Equal(t, codes.DeadlineExceeded, Code(merry.Wrap(context.DeadlineExceeded)))

	// cancelled
	assert.Equal(t, codes.Canceled, Code(merry.Wrap(context.Canceled)))

	// mapped from http code
	assert.Equal(t, codes.Unauthenticated, Code(merry.New("blue", merry.WithHTTPCode(http.StatusUnauthorized))))

	// default
	assert.Equal(t, codes.Unknown, Code(errors.New("blue")))
}

func TestDetailsFromError(t *testing.T) {
	// nil -> nil
	assert.Nil(t, DetailsFromError(nil))

	// error with no details -> nil
	assert.Nil(t, DetailsFromError(errors.New("boring")))

	err := merry.New("blue", merry.WithUserMessage("yikes"), merry.WithFormattedStack([]string{"blue", "red"}))

	assert.Equal(t, []proto.Message{
		&errdetails.LocalizedMessage{Message: "yikes", Locale: "en-US"},
		&errdetails.DebugInfo{StackEntries: []string{"blue", "red"}},
	}, DetailsFromError(err))
}

func TestCodeFromHTTPStatus(t *testing.T) {
	assert.Equal(t, codes.NotFound, CodeFromHTTPStatus(http.StatusNotFound))
	for i := 200; i < 300; i++ {
		assert.Equal(t, codes.OK, CodeFromHTTPStatus(i), "for status code %v", i)
	}
	assert.Equal(t, codes.Unknown, CodeFromHTTPStatus(500))
}
