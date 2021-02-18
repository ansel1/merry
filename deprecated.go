package merry

// Message returns just returns err.Error().  It is here for
// historical reasons.
func Message(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
