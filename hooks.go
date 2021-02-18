package merry

// MaxStackDepth is the maximum number of stackframes on any error.
var MaxStackDepth = 50

var captureStacks = true

// StackCaptureEnabled returns whether stack capturing is enabled
func StackCaptureEnabled() bool {
	return captureStacks
}

// SetStackCaptureEnabled sets stack capturing globally.  Disabling stack capture can increase performance
func SetStackCaptureEnabled(enabled bool) {
	captureStacks = enabled
}

// VerboseDefault no longer has any effect.
// deprecated: see SetVerboseDefault
func VerboseDefault() bool {
	return false
}

// SetVerboseDefault used to control the behavior of the Error() function on errors
// processed by this package.  Error() now always just returns the error's message.
// This setting no longer has any effect.
// deprecated: To print the details of an error, use Details(err), or format the
// error with the verbose flag: fmt.Sprintf("%+v", err)
func SetVerboseDefault(bool) {
}
