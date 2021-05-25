// Package status is a drop-in replacement for the google.golang.org/grpc/status
// package, but is compatible with merry errors.
//
// Errors created by this package are merry errors, and can be augmented with
// additional information, like any other merry error.
//
// Functions which translate errors into a Status, or into a Code are compatible with
// the new error wrapping conventions, using errors.Is and errors.As to extract
// a Status from nested errors.
//
// This package also adds additional utilities for adding Codes or Statuses to an existing error,
// and converting a merry error into a Status.
package status

import (
	"context"
	"fmt"
	"github.com/ansel1/merry/v2"
	"github.com/ansel1/merry/v2/internal"
	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

// Status references google.golang.org/grpc/status
type Status = status.Status

// New returns a Status representing c and msg.
func New(c codes.Code, msg string) *Status {
	return status.New(c, msg)
}

// Newf returns New(c, fmt.Sprintf(format, a...)).
func Newf(c codes.Code, format string, a ...interface{}) *Status {
	return status.Newf(c, format, a...)
}

// Error returns an error representing c and msg.  If c is OK, returns nil.
func Error(c codes.Code, msg string) error {
	return merry.WrapSkipping(New(c, msg).Err(), 1)
}

// Errorf returns Error(c, fmt.Sprintf(format, a...)).
func Errorf(c codes.Code, format string, a ...interface{}) error {
	return merry.WrapSkipping(New(c, fmt.Sprintf(format, a...)).Err(), 1)
}

// ErrorProto returns an error representing s.  If s.Code is OK, returns nil.
func ErrorProto(s *spb.Status) error {
	return merry.WrapSkipping(status.FromProto(s).Err(), 1)
}

// FromProto returns a Status representing s.
func FromProto(s *spb.Status) *Status {
	return status.FromProto(s)
}

// ToError creates an error from a Status.  It is an alternative to Status's Err() method.
// It will translate LocalizedMessage and DebugInfo details on the Status into corresponding
// error properties (user message and formatted stack, respectively).  If the Status doesn't
// contain a DebugInfo detail, it will capture a stack.
func ToError(s *Status) error {
	return merry.WrapSkipping(s.Err(), 1, WithStatusDetails(s))
}

// FromError returns a Status representing err if the error or any of its causes can
// be coerced to a GRPCStatuser with errors.As().  Errors created by this package
// or by google.golang.org/grpc/status have a Status that will be found by this
// function.  A Status can also be associated with an existing error using WithStatus.
//
// If a Status is found, the ok return value will be true.
//
// If no Status is found, ok is false, and a new Status is constructed from the error.
func FromError(err error) (s *Status, ok bool) {
	if err == nil {
		return nil, true
	}

	var statuser GRPCStatuser
	if internal.As(err, &statuser) {
		return statuser.GRPCStatus(), true
	}

	// construct new status from error
	return New(Code(err), err.Error()), false
}

// Convert is a convenience function which removes the need to handle the
// boolean return value from FromError.
func Convert(err error) *Status {
	s, _ := FromError(err)
	return s
}

// FromContextError remains for compatibility with the status package, but
// it does the same thing as Convert/FromError.  The logic for translating
// context errors into appropriate grpc codes is built in to FromError now.
func FromContextError(err error) *Status {
	// the status package used different stuff here, I think because it was
	// written before the go1.13 error enhancements.  Now, this just does the same
	// thing as FromError().
	return Convert(err)
}

// WithStatusDetails translates status details into error attributes:
//
// - LocalizedMessage details are set as the error's user message.
// - DebugInfo details are set as the error's formatted stack.
// - The error's HTTP code is derived from the grpc code.
//
// To construct an error from a status, translated status details
// into error context information:
//
//     err := merry.Wrap(status.Err(), WithStatusDetails(status))
//
func WithStatusDetails(status *Status) merry.Wrapper {
	return merry.WrapperFunc(func(err error, depth int) error {
		if err == nil || status == nil {
			return err
		}

		for _, detail := range status.Details() {
			switch t := detail.(type) {
			case *errdetails.LocalizedMessage:
				message := t.GetMessage()
				if message != "" {
					err = merry.WithUserMessage(message).Wrap(err, depth)
				}
			case *errdetails.DebugInfo:
				entries := t.GetStackEntries()
				if len(entries) > 0 {
					err = merry.WithFormattedStack(entries).Wrap(err, depth)
				}
			}
		}

		return merry.WithHTTPCode(HTTPStatusFromCode(status.Code())).Wrap(err, depth)
	})
}

// WithStatus associates a Status with an error.  This overrides any prior
// Status associated with the error.  FromError/Convert will return this value
// instead of deriving a status from the error.
func WithStatus(status *Status) merry.Wrapper {
	return merry.WrapperFunc(func(err error, _ int) error {
		if status == nil || err == nil {
			return err
		}

		return &grpcStatusError{
			err:    err,
			status: status.Proto(),
		}
	})
}

// WithCode is a merry.Wrapper which associates a GRPC code with the error.  Code() will
// return this value.  This overrides any other mapping of an error
// to a code.
//
// This is the merry-compatible equivalent of the status.Error and
// status.Errorf functions, which don't support error wrapping.
func WithCode(code codes.Code) merry.Wrapper {
	return merry.WrapperFunc(func(err error, depth int) error {
		if err == nil {
			return nil
		}

		status, _ := FromError(err)
		p := status.Proto()
		p.Code = int32(code)
		return WithStatus(FromProto(p)).Wrap(err, depth)
	})
}

// Code returns the grpc response code for an error.  It is
// similar to status.Code(), and should behave identically
// to that function for non-merry errors.  If err is a merry
// error, this supports mapping some merry constructs to grpc codes.
// The rules like a switch statement:
//
// - err is nil: codes.OK
// - grpc.Code was explicitly associated with the error using
//   WithCode(): associated code
// - errors.As(GRPCStatuser): statuser.Status().Code()
// - errors.Is(context.DeadlineExceeded): codes.DeadlineExceeded
// - errors.Is(context.Canceled: codes.Canceled
// - default: CodeFromHTTPStatus(), which defaults to codes.Unknown
func Code(err error) codes.Code {
	var grpcErr GRPCStatuser

	switch {
	case err == nil:
		return codes.OK
	case internal.As(err, &grpcErr):
		return grpcErr.GRPCStatus().Code()
	case internal.Is(err, context.DeadlineExceeded):
		return codes.DeadlineExceeded
	case internal.Is(err, context.Canceled):
		return codes.Canceled
	default:
		return CodeFromHTTPStatus(merry.HTTPCode(err))
	}
}

// DetailsFromError derives status details from context attached to the error:
//
// - if the err has a user message, it will be converted into a LocalizedMessage.
// - if the err has a stack, it will be converted into a DebugInfo.
//
// Returns nil if no details are derived from the error.
func DetailsFromError(err error) []proto.Message {
	var details []proto.Message
	if um := merry.UserMessage(err); um != "" {
		details = append(details, &errdetails.LocalizedMessage{
			Message: um,
		})
	}
	if formattedStack := merry.FormattedStack(err); len(formattedStack) > 0 {
		details = append(details, &errdetails.DebugInfo{
			StackEntries: formattedStack,
		})
	}

	return details
}

// CodeFromHTTPStatus returns a grpc code from an http status code.  It returns
// the inverse of HTTPStatusFromCode, plus some additional HTTP code mappings.
//
// If there is no mapping for the status code, it defaults to OK for status codes
// between 200 and 299, and Unknown for all others.
func CodeFromHTTPStatus(httpStatus int) codes.Code {
	switch httpStatus {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest,
		http.StatusUnprocessableEntity,
		http.StatusNotExtended:
		// bad user input
		return codes.InvalidArgument
	case http.StatusUnauthorized,
		http.StatusNetworkAuthenticationRequired:
		return codes.Unauthenticated
	case http.StatusPaymentRequired,
		http.StatusTooManyRequests,
		http.StatusInsufficientStorage:
		// licensing or throttling limits hit
		return codes.ResourceExhausted
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound,
		http.StatusGone:
		return codes.NotFound
	case http.StatusRequestTimeout:
		return codes.Canceled
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusPreconditionFailed,
		http.StatusLocked:
		return codes.FailedPrecondition
	case http.StatusRequestedRangeNotSatisfiable:
		return codes.OutOfRange
	case http.StatusNotImplemented, http.StatusExpectationFailed:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusFailedDependency:
		return codes.Aborted
	}

	// all 2xx codes are OK
	if httpStatus >= 200 && httpStatus < 300 {
		return codes.OK
	}

	// all other codes map to Unknown
	// This covers all the 5xx codes, where some service along the way really did have an
	// internal error, and all the 4xx codes not handled specifically above.  These 4xx codes
	// typically mean the HTTP client did something wrong (not the server), but in this case
	// the only 4xx codes which fall through to here should be the result of bugs in our
	// own HTTP clients to upstream internal services, not problems with the actually end user
	// input.
	return codes.Unknown
}

// HTTPStatusFromCode converts a gRPC error code into the corresponding HTTP response status.
func HTTPStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		// Note, this deliberately doesn't translate to the similarly named '412 Precondition Failed' HTTP response status.
		return http.StatusBadRequest
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	return http.StatusInternalServerError
}

// GRPCStatuser knows how to return a Status.
type GRPCStatuser interface {
	GRPCStatus() *Status
}

// ensure grpcStatusError implements fmt.Formatter
var _ fmt.Formatter = (*grpcStatusError)(nil)

type grpcStatusError struct {
	err    error
	status *spb.Status
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
	return FromProto(e.status)
}

func (e *grpcStatusError) Format(f fmt.State, verb rune) {
	merry.Format(f, verb, e)
}
