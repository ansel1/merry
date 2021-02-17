package merry

// MaxStackDepth is the maximum number of stackframes on any error.
var MaxStackDepth = 50

var captureStacks = true
var verbose = false

// StackCaptureEnabled returns whether stack capturing is enabled
func StackCaptureEnabled() bool {
	return captureStacks
}

// SetStackCaptureEnabled sets stack capturing globally.  Disabling stack capture can increase performance
func SetStackCaptureEnabled(enabled bool) {
	captureStacks = enabled
}

// VerboseDefault returns the global default for verbose mode.
// When true, e.Error() == Details(e)
// When false, e.Error() == Message(e) + Cause(e)
func VerboseDefault() bool {
	return verbose
}

// SetVerboseDefault sets the global default for verbose mode.
// When true, e.Error() == Details(e)
// When false, e.Error() == Message(e) + Cause(e)
func SetVerboseDefault(b bool) {
	verbose = b
}
