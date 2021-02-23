// +build !go1.13

package merry

import (
	errors "golang.org/x/xerrors"
)

// If using <go1.13, polyfill errors.Is/As with golang.org/x/xerrors
// xerrors can be removed once <go1.12 support is dropped

var is = errors.Is

var as = errors.As

var unwrap = errors.Unwrap