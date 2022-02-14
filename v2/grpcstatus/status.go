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
// This package also adds additional utilities for adding Codes to an existing error,
// and converting a merry error into a Status.
package status

import (
	"context"
	"errors"
	"fmt"
	"github.com/ansel1/merry/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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
	if errors.As(err, &statuser) {
		grpcStatus := statuser.GRPCStatus()

		// check whether the code was overridden via WithCode
		if code, ok := lookupCode(err); ok && code != grpcStatus.Code() {
			stProto := grpcStatus.Proto()
			stProto.Code = int32(code)
			grpcStatus = FromProto(stProto)
		}

		return grpcStatus, true
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

// WithCode is a merry.Wrapper which associates a GRPC code with the error.
// Code() will return this value.
func WithCode(code codes.Code) merry.Wrapper {
	return merry.WithValue(errValueKeyCode, code)
}

// Code returns the grpc response code for an error.  It is
// similar to status.Code(), and should behave identically
// to that function for non-merry errors.  If err is a merry
// error, this supports mapping some merry constructs to grpc codes.
// The rules like a switch statement:
//
// - err is nil: codes.OK
// - code previously set with WithCode()
// - errors.As(GRPCStatuser): return code from Status
// - errors.Is(context.DeadlineExceeded): codes.DeadlineExceeded
// - errors.Is(context.Canceled: codes.Canceled
// - default: CodeFromHTTPStatus(), which defaults to codes.Unknown
func Code(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	if code, ok := lookupCode(err); ok {
		return code
	}

	var grpcErr GRPCStatuser

	switch {
	case errors.As(err, &grpcErr):
		return grpcErr.GRPCStatus().Code()
	case errors.Is(err, context.DeadlineExceeded):
		return codes.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		return codes.Canceled
	default:
		return CodeFromHTTPStatus(merry.HTTPCode(err))
	}
}

func lookupCode(err error) (codes.Code, bool) {
	if codeVal, ok := merry.Lookup(err, errValueKeyCode); ok {
		code, ok := codeVal.(codes.Code)
		return code, ok
	}
	return codes.OK, false
}

// DefaultLocalizedMessageLocale is the value used when encoding a merry.UserMessage()
// to a errdetails.LocalizedMessage.
var DefaultLocalizedMessageLocale = "en-US"

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
			Locale:  DefaultLocalizedMessageLocale,
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
// the inverse of github.com/grpc-ecosystem/grpc-gateway/v2/runtime.HTTPStatusFromCode,
// plus some additional HTTP code mappings.
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

// GRPCStatuser knows how to return a Status.
type GRPCStatuser interface {
	GRPCStatus() *Status
}

// errValueKey is a private type for merry error value keys
type errValueKey int

// errValueKeyCode is a private key for storing a grpc code as a merry error value
const errValueKeyCode = iota
