package richerrors

import (
	"bytes"
	goerr "github.com/go-errors/errors"
)

// Returns the error's stacktrace as a string formatted
// the same way as golangs runtime package.
// If e has no stacktrace, returns an empty string.
func Stacktrace(e error) string {
	if e, ok := e.(Stacker); ok {
		s := e.Stack()
		if len(s) > 0 {
			buf := bytes.Buffer{}
			for _, fp := range s {
				sf := goerr.NewStackFrame(fp)
				buf.WriteString(sf.String())
			}
			return buf.String()
		}
	}
	return ""
}

// Returns e.Error() and e's stacktrace.
// If e has no stacktrace, this is identical to e.Error()
func Details(e error) string {
	msg := e.Error()
	s := Stacktrace(e)
	if s != "" {
		msg += "\n" + s
	}
	return msg
}
