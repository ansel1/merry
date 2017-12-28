package merry

import (
	"bytes"
	"fmt"

	"runtime"
)

// Location returns zero values if e has no stacktrace
func Location(e error) (file string, line int) {
	s := Stack(e)
	if len(s) > 0 {
		fnc := runtime.FuncForPC(s[0])
		if fnc != nil {
			return fnc.FileLine(s[0])
		}
	}
	return "", 0
}

// SourceLine returns the string representation of
// Location's result or an empty string if there's
// no stracktrace.
func SourceLine(e error) string {
	file, line := Location(e)
	if line != 0 {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return ""
}

// Stacktrace returns the error's stacktrace as a string formatted
// the same way as golangs runtime package.
// If e has no stacktrace, returns an empty string.
func Stacktrace(e error) string {
	s := Stack(e)
	if len(s) > 0 {
		buf := bytes.Buffer{}
		for _, fp := range s {
			fnc := runtime.FuncForPC(fp)
			if fnc != nil {
				f, l := fnc.FileLine(fp)
				buf.WriteString(fnc.Name())
				buf.WriteString(fmt.Sprintf("\n\t%s:%d\n", f, l))
			}
		}
		return buf.String()
	}
	return ""
}

// Details returns e.Error() and e's stacktrace and user message, if set.
func Details(e error) string {
	if e == nil {
		return ""
	}
	msg := Message(e)
	userMsg := UserMessage(e)
	if userMsg != "" {
		msg = fmt.Sprintf("%s\n\nUser Message: %s", msg, userMsg)
	}
	s := Stacktrace(e)
	if s != "" {
		msg += "\n\n" + s
	}
	return msg
}
