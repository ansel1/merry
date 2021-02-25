// +build !go1.13

package internal

import (
	errors "golang.org/x/xerrors"
)

// If using <go1.13, polyfill errors.Is/As with golang.org/x/xerrors
// xerrors can be removed once <go1.12 support is dropped

// Unwrap backfill
var Unwrap = errors.Unwrap
