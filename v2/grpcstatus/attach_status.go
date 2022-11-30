package status

import (
	"fmt"
	"github.com/ansel1/merry/v2"
	"google.golang.org/grpc/status"
)

// AttachStatus associates a Status with an error.  status.FromError and
// status.Convert will return this value instead of deriving a status from the error.
//
// This should be used by GRPC handlers which want to craft a specific status.Status
// to return.  The result of this function should be returned by the handler *without
// any further error wrapping* (because grpc does not support error wrapping), like this:
//
//	func MyHandler(ctx context.Context, req *MyReq) (*MyResp, error) {
//	  resp, err := handle(ctx, req)
//	  if err != nil {
//	    sts := status.Convert(err)
//	    // customize the status
//	    return nil, status.AttachStatus(err, sts)
//	  }
//	}
//
// # tl;dr
//
// This would typically be done like this:
//
//	func MyHandler(ctx context.Context, req *MyReq) (*MyResp, error) {
//	  resp, err := handle(ctx, req)
//	  if err != nil {
//	    sts := status.Convert(err)
//	    // customize the status
//	    return nil, status.Err()
//	  }
//	}
//
// The trouble with that is that the original error is lost.  If you have interceptors
// which log errors, they will never see the original error, which might have had all
// sorts of interesting information in them.  You also can't do this:
//
//	return merry.Wrap(sts.Err(), merry.WithCause(err))
//
// ...because the grpc package doesn't handle wrapped errors.  You cannot wrap
// sts.Err() any further, or the status will be lost.  According to this [PR],
// it seems this is intentional, and wrapped errors may never be supported.
//
// This is intentionally *not* a merry wrapper function, because the result
// of this function should never be wrapped any further.  It needs to be returned
// as-is from the handler in order for the grpc code to find your status.Status
// and return it to the client.
//
// [PR]: https://github.com/grpc/grpc-go/pull/4091
func AttachStatus(err error, status *Status) error {
	if status == nil || err == nil {
		return err
	}

	return &grpcStatusError{
		err:    err,
		status: status,
	}
}

// ensure grpcStatusError implements fmt.Formatter
var _ fmt.Formatter = (*grpcStatusError)(nil)

type grpcStatusError struct {
	err    error
	status *status.Status
}

func (e *grpcStatusError) Error() string {
	return e.err.Error()
}

func (e *grpcStatusError) String() string {
	return e.Error()
}

func (e *grpcStatusError) Unwrap() error {
	return e.err
}

func (e *grpcStatusError) GRPCStatus() *Status {
	return e.status
}

func (e *grpcStatusError) Format(f fmt.State, verb rune) {
	merry.Format(f, verb, e)
}
