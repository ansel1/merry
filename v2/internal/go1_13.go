// +build go1.13

package internal

import "errors"

// If using >=go1.13, golang.org/x/xerrors is not needed.
// xerrors can be removed once <go1.12 support is dropped

// Is alias
var Is = errors.Is

// As alias
var As = errors.As

// Unwrap alias
var Unwrap = errors.Unwrap
