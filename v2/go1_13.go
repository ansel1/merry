// +build go1.13

package merry

import "errors"

// If using >=go1.13, golang.org/x/xerrors is not needed.
// xerrors can be removed once <go1.12 support is dropped

var is = errors.Is

var as = errors.As

var unwrap = errors.Unwrap