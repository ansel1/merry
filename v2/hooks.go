package merry

var hooks []Wrapper

// AddHooks installs a global set of Wrappers which are applied to every error processed
// by this package.  They are applied before any other Wrappers or stack capturing are
// applied.  Hooks can add additional wrappers to errors, or translate annotations added
// by other error libraries into merry annotations.
//
// This function is not thread safe, and should only be called very early in program
// initialization.
func AddHooks(hook ...Wrapper) {
	hooks = append(hooks, hook...)
}

// ClearHooks removes all installed hooks.
//
// This function is not thread safe, and should only be called very early in program
// initialization.
func ClearHooks() {
	hooks = nil
}
